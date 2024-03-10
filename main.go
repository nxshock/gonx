package main

import (
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

	app, err := newApp(config)
	if err != nil {
		slog.Error("Failed to start app", slog.String("err", err.Error()))
		os.Exit(1)
	}

	err = app.restartTlsListener()
	if err != nil {
		slog.Error("Failed to start TLS listener", slog.String("err", err.Error()))
		os.Exit(1)
	}

	go func() {
		slog.Debug("Starting HTTP listener", slog.String("addr", config.HttpListenAddr))

		smux := http.NewServeMux()
		smux.Handle(defaultAcmeChallengePath, http.FileServer(http.Dir(config.AcmeChallengePath)))
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

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		for {
			<-c
			slog.Debug("TLS keys reload requested")

			err := app.reloadConfig(configFilePath)
			if err != nil {
				slog.Error("failed to reload TLS keys", slog.String("err", err.Error()))
			}
			slog.Debug("Reloading TLS keys completed")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	slog.Debug("Interrupt signal received.")
}
