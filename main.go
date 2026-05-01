package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := setupLogger()
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("proxy failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	apiKey := parseAPIKey()
	h := newNewsHandler(apiKey, tpl, logger)

	port := getPort()
	mux := http.NewServeMux()
	s := http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  readR,
		WriteTimeout: writeR,
		IdleTimeout:  keepA,
	}

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/search", h.search)
	mux.HandleFunc("/", h.index)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			logger.Error("error starting server", slog.Any("error", err))
			os.Exit(1)
		}
	}()
	logger.Info("server started", slog.String("port", port))

	<-ctx.Done()
	logger.Info("signal closing server received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", slog.Any("error", err))
	}
	logger.Info("server shutdown gracefully")
	return nil
}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Value.Kind() == slog.KindDuration {
				return slog.String(a.Key, fmt.Sprintf("%dms", a.Value.Duration().Milliseconds()))
			}
			return a
		},
	}))
}
