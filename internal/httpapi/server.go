package httpapi

import (
	"net/http"
	"time"
)

const (
	readTimeout       = 7 * time.Second   // max time to read request from the client
	readHeaderTimeout = 5 * time.Second   // max time to read request headers
	writeTimeout      = 10 * time.Second  // max time to write response to the client
	idleTimeout       = 120 * time.Second // max time for connections using TCP Keep-Alive
)

// NewServer returns an http.Server with predefined timeouts.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
