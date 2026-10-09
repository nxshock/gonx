package main

import _ "embed"

const (
	defaultConfigPath  = "/etc/gonx.conf"
	defaultTlsKeysPath = "/etc/letsencrypt/live"

	defaultCertFileName = "fullchain.pem"
	defaultKeyFileName  = "privkey.pem"

	defaultAcmeChallengePath = "/.well-known/acme-challenge/"
)

var (
	//go:embed defaultErrorPage.html
	defaultErrorPage string
)
