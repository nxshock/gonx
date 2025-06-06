# gonx

Simple reverse proxy server.

## Features:

* Simple TCP redirection
* Simple static file server

## Usage

1. Install package
2. Edit config in `/etc/gonx.conf`
3. Start application with systemd:
   
   `systemctl start gonx.service`

## Config example

```toml
LogLevel = "DEBUG"                         # Log level (DEBUG, INFO, WARN, ERROR)
TlsKeysDir = "/etc/letsencrypt/live"       # Path to TLS-certificates generated by Certbot
TlsListenAddr = ":443"                     # TLS listen address
HttpListenAddr = ":80"                     # HTTP listen address
AcmeChallengePath = "/var/lib/letsencrypt" # Path for ACME challenge files

# Map of hostname -> redirect URL
[TLS]
"git.host.com"  = "tcp://127.0.0.1:8001"           # TCP redirect
"unix.host.com" = "unix:///var/lib/app/app.socket" # serve unix socket
"www.host.com"  = "file:///srv/http"               # simple static file server from `/srv/http`
```
