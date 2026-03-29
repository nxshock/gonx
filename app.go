package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/nxshock/simplelog"
)

type App struct {
	Config         *Config
	configFilePath string

	logger *simplelog.Logger

	tlsListener net.Listener

	// Current opened connections count
	OpenedConnections int64

	// Total processed conections count
	TotalConnectionsCount int64
}

func newApp() *App {
	return new(App)
}

func (app *App) LoadConfig(configFilePath string) error {
	config := new(Config)

	_, err := toml.DecodeFile(configFilePath, &config)
	if err != nil {
		return err
	}

	config.proxyRules = make(HostMapping)
	for inputUrlStr, outputUrlStr := range config.TLS {
		err = config.proxyRules.Add(app, inputUrlStr, outputUrlStr)
		if err != nil {
			return err
		}
	}

	app.Config = config
	app.configFilePath = configFilePath

	err = app.Config.initTls()
	if err != nil {
		return fmt.Errorf("failed to load TLS keys: %w", err)
	}

	return nil
}

func (app *App) reloadConfig(configFilePath string) error {
	err := app.LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	logger.Debug("Deactivating TLS listener on", app.Config.TlsListenAddr)
	err = app.tlsListener.Close()
	if err != nil {
		return fmt.Errorf("failed to close TLS listener: %v", err)
	}

	err = app.restartTlsListener()
	if err != nil {
		return fmt.Errorf("failed to restart TLS listener: %v", err)
	}

	return nil
}

func (app *App) restartTlsListener() error {
	logger.Debug("Starting TLS listener on", app.Config.TlsListenAddr)

	tlsListener, err := tls.Listen("tcp", app.Config.TlsListenAddr, app.Config.tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to open tls listener: %v", err)
	}

	app.tlsListener = tlsListener

	go func() {
		for {
			conn, err := tlsListener.Accept()
			if err != nil {
				logger.Error("failed to receive connection:", err) // TODO: drop error on closing TLS listener
				break
			}
			logger.Debugln("incoming connection from", conn.RemoteAddr().String())

			go func() { _ = handleTlsConn(conn.(*tls.Conn), app.Config.proxyRules) }()
		}
	}()

	return nil
}

func (app *App) start() error {
	if len(app.Config.TLS) > 0 {
		err := app.restartTlsListener()
		if err != nil {
			return fmt.Errorf("Failed to start TLS listener: %v", err)
		}
	} else {
		logger.Warn("TLS listener does not started because TLS redirection rules is empty")
	}

	go func() {
		logger.Debug("Starting HTTP listener on ", app.Config.HttpListenAddr)

		smux := http.NewServeMux()
		smux.Handle(defaultAcmeChallengePath, http.FileServer(http.Dir(app.Config.AcmeChallengePath)))
		smux.HandleFunc("/gonx/stats", app.handleStats())
		smux.HandleFunc("/", handleDefault)

		httpServer := http.Server{Handler: smux}
		httpServer.Addr = app.Config.HttpListenAddr
		err := httpServer.ListenAndServe()
		if err != nil {
			logger.Errorln("Failed to start HTTP server:", err.Error())
			os.Exit(1)
		}
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		for {
			<-c
			logger.Debug("TLS keys reload requested")

			err := app.reloadConfig(app.configFilePath)
			if err != nil {
				logger.Error("failed to reload TLS keys: ", err.Error())
			}
			logger.Debug("Reloading TLS keys completed")
		}
	}()

	return nil
}

func (app *App) handleStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if host != "127.0.0.1" {
			http.Error(w, "not allowed", http.StatusForbidden)
			return
		}

		enc := json.NewEncoder(w)
		enc.SetIndent("", "\t")
		enc.Encode(app)
	}
}

func (app *App) handleListener(conn *tls.Conn, outputUrl *url.URL) {
	logger.Debugf("%s -> %s", conn.RemoteAddr(), outputUrl.Host+outputUrl.Path)

	atomic.AddInt64(&app.OpenedConnections, 1)
	atomic.AddInt64(&app.TotalConnectionsCount, 1)

	defer func() {
		conn.Close()
		atomic.AddInt64(&app.OpenedConnections, -1)
	}()

	c, err := net.Dial(outputUrl.Scheme, outputUrl.Host+outputUrl.Path)
	if err != nil {
		fmt.Fprintf(conn, "HTTP/1.1 500 Internal Server Error\r\nConnection: Close\r\nContent-Type: text/plain\r\n\r\n%s", err)
		return
	}
	defer c.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()

		io.Copy(conn, c)
		c.Close()
	}()

	go func() {
		defer wg.Done()

		io.Copy(c, conn)
		c.Close() // TODO: why write thread's `io.Copy` does not closes when read thread is closed?
	}()

	wg.Wait()
}
