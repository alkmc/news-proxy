package config

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"
)

type (
	// Config holds application configuration.
	Config struct {
		Server   Server
		NewsAPI  NewsAPI
		LogLevel slog.Level
	}
	// Server holds HTTP server configuration.
	Server struct {
		Port              string
		ReadTimeout       time.Duration
		ReadHeaderTimeout time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
		ShutdownTimeout   time.Duration
	}
	// NewsAPI holds upstream NewsAPI configuration.
	NewsAPI struct {
		BaseURL      string
		APIKey       string
		PageSize     int
		MaxResults   int
		FetchTimeout time.Duration
	}
)

// Load reads configuration from environment variables.
func Load() (Config, error) {
	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		return Config{}, errors.New("NEWS_API_KEY environment variable must be set")
	}

	var logLevel slog.Level // zero value is INFO
	if s := os.Getenv("LOG_LEVEL"); s != "" {
		if err := logLevel.UnmarshalText([]byte(s)); err != nil {
			return Config{}, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
	}

	return Config{
		Server: Server{
			Port:              cmp.Or(os.Getenv("PORT"), "8080"),
			ReadTimeout:       7 * time.Second,   // max time to read request from the client
			ReadHeaderTimeout: 5 * time.Second,   // max time to read request headers
			WriteTimeout:      10 * time.Second,  // max time to write response to the client
			IdleTimeout:       120 * time.Second, // max time for connections using TCP Keep-Alive
			ShutdownTimeout:   10 * time.Second,  // max time to complete tasks before shutdown
		},
		NewsAPI: NewsAPI{
			BaseURL:      "https://newsapi.org",
			APIKey:       apiKey,
			PageSize:     10,               // articles per NewsAPI page
			MaxResults:   100,              // NewsAPI free tier limit
			FetchTimeout: 10 * time.Second, // max time to fetch data from NewsAPI
		},
		LogLevel: logLevel,
	}, nil
}

// Address returns the HTTP server listen address.
func (s Server) Address() string {
	return net.JoinHostPort("", s.Port)
}
