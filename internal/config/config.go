package config

import (
	"errors"
	"net"
	"os"
	"time"
)

const (
	ReadTimeout       = 7 * time.Second   // max time to read request from the client
	ReadHeaderTimeout = 5 * time.Second   // max time to read request headers
	WriteTimeout      = 10 * time.Second  // max time to write response to the client
	IdleTimeout       = 120 * time.Second // max time for connections using TCP Keep-Alive
	ShutdownTimeout   = 10 * time.Second  // max time to complete tasks before shutdown
	defaultPort       = "3000"
	PageSize          = 10
	BaseURL           = "https://newsapi.org"
)

func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return net.JoinHostPort("", port)
}

func GetAPIKey() (string, error) {
	apiKey := os.Getenv("NEWS_API_KEY")

	if apiKey == "" {
		return "", errors.New("NEWS_API_KEY environment variable must be set")
	}
	return apiKey, nil
}
