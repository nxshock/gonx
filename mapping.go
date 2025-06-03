package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
)

type ProxyDirection struct {
	Output *url.URL

	listener *Listener
}

type HostMapping map[string]ProxyDirection // hostName -> rule

func (h HostMapping) Add(host, outputUrlStr string) error {
	outputUrl, err := url.Parse(outputUrlStr)
	if err != nil {
		return err
	}

	pd := ProxyDirection{outputUrl, NewListener()}

	switch outputUrl.Scheme {
	case "file":
		server := http.Server{Handler: http.FileServer(http.Dir(outputUrl.Path))}
		go server.Serve(pd.listener)
	case "tcp":
		go func(pd ProxyDirection) {
			for {
				conn, err := pd.listener.Accept()
				if err != nil {
					slog.Debug(err.Error())
					continue
				}
				go handleTcp(conn.(*tls.Conn), pd.Output)
			}
		}(pd)
	case "unix":
		go func(pd ProxyDirection) {
			for {
				conn, err := pd.listener.Accept()
				if err != nil {
					slog.Debug(err.Error())
					continue
				}
				go handleUnix(conn.(*tls.Conn), pd.Output)
			}
		}(pd)
	default:
		return fmt.Errorf("unknown output protocol: %v", outputUrl.Scheme)
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

func handleTcp(conn *tls.Conn, outputUrl *url.URL) {
	slog.Debug(fmt.Sprintf("%s -> %s", conn.RemoteAddr(), outputUrl.Host))

	c, err := net.Dial(outputUrl.Scheme, outputUrl.Host)
	if err != nil {
		writeError(conn, err)
		conn.Close()
		return
	}
	defer c.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, _ = io.Copy(conn, c)
	}()

	go func() {
		defer wg.Done()

		_, _ = io.Copy(c, conn)
	}()

	wg.Wait()
}

func handleUnix(conn *tls.Conn, outputUrl *url.URL) {
	slog.Debug(fmt.Sprintf("%s -> %s", conn.RemoteAddr(), outputUrl.Host+outputUrl.Path))

	c, err := net.Dial(outputUrl.Scheme, outputUrl.Host+outputUrl.Path)
	if err != nil {
		writeError(conn, err)
		conn.Close()
		return
	}
	defer c.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, _ = io.Copy(conn, c)
	}()

	go func() {
		defer wg.Done()

		_, _ = io.Copy(c, conn)
	}()

	wg.Wait()
}

func writeError(w io.Writer, err error) {
	fmt.Fprintf(w, "HTTP/1.1 500 Internal Server Error\r\nConnection: Close\r\nContent-Type: text/plain\r\n\r\n%s", err)
}
