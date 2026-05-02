package api

import (
	"net/http"

	"github.com/alkmc/firstGoApp/internal/config"
)

// NewServer creates a configured http.Server instance with timeouts from global config.
func NewServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}
}
