package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/nxshock/simplelog"
)

var logger *simplelog.Logger

func init() {
	logger = simplelog.NewLogger(os.Stderr)

	// Disable timestamps if running in systemd
	if os.Getppid() == 1 {
		logger.TimeFormat = ""
	}
}

func main() {
	configFilePath := defaultConfigPath
	if len(os.Args) > 1 {
		configFilePath = os.Args[1]
	}

	app := newApp()

	err := app.LoadConfig(configFilePath)
	if err != nil {
		logger.Fatalln("Failed to load config:", err)
	}

	logger.Level = simplelog.LogLevel(app.Config.LogLevel)

	err = app.start()
	if err != nil {
		logger.Fatalln("Failed to start app:", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Debug("Interrupt signal received")
}
