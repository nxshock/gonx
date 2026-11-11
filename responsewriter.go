package main

import (
	"fmt"
	"io"
	"net/http"
)

type responseWriter struct {
	header http.Header
	w      io.Writer

	wroteHeader bool
}

func newResponseWriter(w io.Writer) *responseWriter {
	return &responseWriter{
		header: make(http.Header),
		w:      w}
}

func (w *responseWriter) Header() http.Header { return w.header } // TODO: implement

func (w *responseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
		w.wroteHeader = true
	}

	return w.w.Write(p)
}
func (w *responseWriter) WriteHeader(statusCode int) {
	fmt.Fprintf(w.w, "HTTP/1.1 %d %s\r\nConnection: Close\r\nContent-Type: text/html\r\n\r\n", statusCode, http.StatusText(statusCode)) // TODO: implement
	w.wroteHeader = true
}
