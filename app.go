package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
)

type App struct {
	config      *Config
	tlsListener net.Listener
}

func newApp(config *Config) (*App, error) {
	err := config.initTls()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS keys: %v", err)
	}

	app := &App{config: config}

	return app, nil
}

func (app *App) reloadConfig(configFilePath string) error {
	config, err := LoadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	err = config.initTls()
	if err != nil {
		return fmt.Errorf("failed to load TLS keys: %v", err)
	}

	app.config = config

	slog.Debug("Deactivating TLS listener", slog.String("addr", app.config.TlsListenAddr))
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
	slog.Debug("Starting TLS listener", slog.String("addr", app.config.TlsListenAddr))

	tlsListener, err := tls.Listen("tcp", app.config.TlsListenAddr, app.config.tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to open tls listener: %v", err)
	}

	app.tlsListener = tlsListener

	go func() {
		for {
			conn, err := tlsListener.Accept()
			if err != nil {
				slog.Error("failed to receive connection", slog.String("err", err.Error())) // TODO: drop error on closing TLS listener
				break
			}
			slog.Debug("incoming connection", slog.String("RemoteAddr", conn.RemoteAddr().String()))

			go func() { _ = handleTlsConn(conn.(*tls.Conn), app.config.proxyRules) }()
		}
	}()

	return nil
}
