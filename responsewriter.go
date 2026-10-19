package main

import (
	"fmt"
	"io"
	"net/http"
)

type responseWriter struct {
	header http.Header
	w      io.Writer
}

func newResponseWriter(w io.Writer) *responseWriter {
	return &responseWriter{
		header: make(http.Header),
		w:      w}
}

func (w responseWriter) Header() http.Header { return w.header } // TODO: implement

func (w responseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}
func (w responseWriter) WriteHeader(statusCode int) {
	fmt.Fprintf(w.w, "HTTP/1.1 %d Internal Server Error\r\nConnection: Close\r\nContent-Type: text/html\r\n\r\n", statusCode)
}
