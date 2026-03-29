package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
)

type ProxyDirection struct {
	Output *url.URL

	listener *Listener
}

type HostMapping map[string]ProxyDirection // hostName -> rule

func (h HostMapping) Add(app *App, host, outputUrlStr string) error {
	outputUrl, err := url.Parse(outputUrlStr)
	if err != nil {
		return err
	}

	pd := ProxyDirection{outputUrl, NewListener()}

	switch outputUrl.Scheme {
	case "file":
		server := http.Server{Handler: http.FileServer(http.Dir(outputUrl.Path))}
		go server.Serve(pd.listener)
	case "tcp", "unix":
		go func(pd ProxyDirection) {
			for {
				conn, err := pd.listener.Accept()
				if err != nil {
					logger.Error(err.Error())
					continue
				}
				go app.handleListener(conn.(*tls.Conn), pd.Output)
			}
		}(pd)
	default:
		return fmt.Errorf("unknown output protocol: %s", outputUrl.Scheme)
	}

	h[host] = pd

	return nil
}

func handleTlsConn(conn *tls.Conn, hosts HostMapping) error {
	err := conn.Handshake()
	if err != nil {
		return fmt.Errorf("handshake error: %v", err)
	}

	hostName := conn.ConnectionState().ServerName
	proxyDirection, exists := hosts[hostName]
	if !exists {
		return fmt.Errorf("requested host not found: %s", hostName)
	}

	proxyDirection.listener.Add(conn)

	return nil
}
