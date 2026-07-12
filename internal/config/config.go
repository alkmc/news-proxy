package config

import (
	"cmp"
	"errors"
	"net"
	"os"
	"time"
)

const (
	ShutdownTimeout = 10 * time.Second // max time to complete tasks before shutdown
	FetchTimeout    = 10 * time.Second // max time to fetch data from NewsAPI
	defaultPort     = "8080"

	PageSize   = 10                    // articles per NewsAPI page
	MaxResults = 100                   // NewsAPI free tier limit
	BaseURL    = "https://newsapi.org" // NewsAPI root URL
)

// ListenAddr returns the listen address.
func ListenAddr() string {
	port := cmp.Or(os.Getenv("PORT"), defaultPort)
	return net.JoinHostPort("", port)
}

// APIKey reads NEWS_API_KEY from the environment.
func APIKey() (string, error) {
	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		return "", errors.New("NEWS_API_KEY environment variable must be set")
	}
	return apiKey, nil
}
