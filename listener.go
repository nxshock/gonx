package main

import (
	"net"
)

// Listener implements net.Listener and additional Add(net.Conn) method.
type Listener struct {
	c chan net.Conn
}

func NewListener() *Listener {
	c := make(chan net.Conn)

	return &Listener{c: c}
}

func (m *Listener) Add(conn net.Conn) {
	m.c <- conn
}

func (m *Listener) Accept() (net.Conn, error) {
	return <-m.c, nil
}

func (m *Listener) Close() error {
	close(m.c)

	return nil
}

func (m *Listener) Addr() net.Addr {
	return nil
}
