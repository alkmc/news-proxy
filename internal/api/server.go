package api

import (
	"net/http"
	"time"
)

const (
	ReadTimeout       = 7 * time.Second   // max time to read request from the client
	ReadHeaderTimeout = 5 * time.Second   // max time to read request headers
	WriteTimeout      = 10 * time.Second  // max time to write response to the client
	IdleTimeout       = 120 * time.Second // max time for connections using TCP Keep-Alive
)

// NewServer returns an http.Server with predefined timeouts.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}
}
