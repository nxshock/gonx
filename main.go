package main

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lmittmann/tint"
)

func main() {
	configFilePath := defaultConfigPath
	if len(os.Args) > 1 {
		configFilePath = os.Args[1]
	}

	config, err := LoadConfig(configFilePath)
	if err != nil {
		slog.Error("Failed to load config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: config.LogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		}}))
	slog.SetDefault(logger)

	err = config.initTls()
	if err != nil {
		slog.Error("init tls error", slog.String("err", err.Error()))
		os.Exit(1)
	}

	go func() {
		slog.Debug("Starting TLS listener", slog.String("addr", config.TlsListenAddr))

		listener, err := tls.Listen("tcp", config.TlsListenAddr, config.tlsConfig)
		if err != nil {
			slog.Error("Failed to open tls listener", slog.String("err", err.Error()))
			os.Exit(1)
		}

		for {
			conn, err := listener.Accept()
			if err != nil {
				slog.Debug("incoming connection failed", slog.String("err", err.Error()))
				continue
			}
			slog.Debug("incoming connection", slog.String("RemoteAddr", conn.RemoteAddr().String()))

			go func() { _ = handleTlsConn(conn.(*tls.Conn), config.proxyRules) }()
		}
	}()

	go func() {
		slog.Debug("Starting HTTP listener", slog.String("addr", config.HttpListenAddr))

		smux := http.NewServeMux()
		smux.Handle(defaultAcmeChallengePath, http.StripPrefix(defaultAcmeChallengePath, http.FileServer(http.Dir(config.AcmeChallengePath))))
		smux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
		})
		httpServer := http.Server{Handler: smux}
		httpServer.Addr = config.HttpListenAddr
		err := httpServer.ListenAndServe()
		if err != nil {
			slog.Error("Failed to start HTTP server", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	slog.Debug("Interrupt signal received.")
}
