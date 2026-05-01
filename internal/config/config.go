package config

import (
	"errors"
	"flag"
	"net"
	"os"
	"time"
)

const (
	ReadTimeout     = 5 * time.Second   // max time to read request from the client
	WriteTimeout    = 10 * time.Second  // max time to write response to the client
	IdleTimeout     = 120 * time.Second // max time for connections using TCP Keep-Alive
	ShutdownTimeout = 10 * time.Second  // max time to complete tasks before shutdown
	defaultPort     = "3000"
	PageSize        = 10
)

func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	return net.JoinHostPort("", port)
}

func ParseAPIKey() (string, error) {
	apiKey := flag.String("apiKey", "", "newsapi.org access key")
	flag.Parse()

	if *apiKey == "" {
		return "", errors.New("apiKey must be set")
	}
	return *apiKey, nil
}
