package httpapi

import (
	"net/http"
	"time"
)

// ServerTimeouts holds HTTP server timeout values.
type ServerTimeouts struct {
	Read       time.Duration
	ReadHeader time.Duration
	Write      time.Duration
	Idle       time.Duration
}

// NewServer builds the HTTP server.
func NewServer(addr string, handler http.Handler, timeouts ServerTimeouts) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       timeouts.Read,
		ReadHeaderTimeout: timeouts.ReadHeader,
		WriteTimeout:      timeouts.Write,
		IdleTimeout:       timeouts.Idle,
	}
}
