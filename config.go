package main

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	// Log level
	LogLevel slog.Level

	// Path to TLS-certificates generated by Certbot
	TlsKeysDir string

	// TLS listen address
	TlsListenAddr string

	// HTTP listen address
	HttpListenAddr string

	// Map of hostname -> redirect URL
	TLS map[string]string

	// Acme path
	AcmeChallengePath string

	// Parsed list of servers
	proxyRules HostMapping

	// loaded TLS keys
	tlsConfig *tls.Config
}

func LoadConfig(configFilePath string) (*Config, error) {
	config := new(Config)

	_, err := toml.DecodeFile(configFilePath, &config)
	if err != nil {
		return nil, err
	}

	config.proxyRules = make(HostMapping)
	for inputUrlStr, outputUrlStr := range config.TLS {
		err = config.proxyRules.Add(inputUrlStr, outputUrlStr)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

func (c *Config) initTls() error {
	c.tlsConfig = new(tls.Config)

	for hostName := range c.proxyRules {
		slog.Debug("reading tls key", slog.String("host", hostName))
		certFilePath := filepath.Join(c.TlsKeysDir, hostName, defaultCertFileName)
		keyFilePath := filepath.Join(c.TlsKeysDir, hostName, defaultKeyFileName)

		cert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
		if err != nil {
			return fmt.Errorf("read tls files error: %v", err)
		}

		c.tlsConfig.Certificates = append(c.tlsConfig.Certificates, cert)
	}

	return nil
}