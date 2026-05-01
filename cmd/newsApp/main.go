package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alkmc/firstGoApp/internal/api"
	"github.com/alkmc/firstGoApp/web"
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
	apiKey, err := api.ParseAPIKey()
	if err != nil {
		return err
	}
	tpl, err := template.ParseFS(web.FS, "template/index.html")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	h := api.NewNewsHandler(apiKey, tpl, logger)

	port := api.GetPort()
	mux := http.NewServeMux()
	s := http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  api.ReadTimeout,
		WriteTimeout: api.WriteTimeout,
		IdleTimeout:  api.IdleTimeout,
	}

	mux.Handle("/static/", http.FileServer(http.FS(web.FS)))

	mux.HandleFunc("/search", h.Search)
	mux.HandleFunc("/", h.Index)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), api.ShutdownTimeout)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
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
