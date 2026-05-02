package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alkmc/firstGoApp/internal/api"
	"github.com/alkmc/firstGoApp/internal/config"
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
	apiKey, err := config.GetAPIKey()
	if err != nil {
		return err
	}
	tpl, err := template.ParseFS(web.FS, "template/index.html")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	client, err := api.NewClient(config.BaseURL, apiKey, config.PageSize, logger)
	if err != nil {
		return fmt.Errorf("failed to create news client: %w", err)
	}
	h := api.NewNewsHandler(client, tpl, logger)

	port := config.GetPort()
	mux := http.NewServeMux()

	s := http.Server{
		Addr:              port,
		Handler:           api.LogMD(logger)(mux),
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}

	mux.Handle("GET /static/", api.CacheMiddleware(http.FileServer(http.FS(web.FS))))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- s.ListenAndServe()
	}()
	logger.Info("server started", slog.String("port", port))

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server listen failed: %w", err)
		}
	case <-ctx.Done():
		logger.Info("signal closing server received")
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), config.ShutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
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
