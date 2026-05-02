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

	client, err := api.NewClient(api.Config{
		BaseURL:    config.BaseURL,
		APIKey:     apiKey,
		PageSize:   config.PageSize,
		MaxResults: config.MaxResults,
		Timeout:    config.FetchTimeout,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create news client: %w", err)
	}
	h := api.NewNewsHandler(client, tpl, logger)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", api.CacheMiddleware(http.FileServer(http.FS(web.FS))))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)

	port := config.GetPort()
	srv := api.NewServer(port, api.LogMD(logger)(mux))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
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
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	logger.Info("server shutdown gracefully")
	return nil
}

func setupLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: loggerReplaceAttrs,
	}))
}

func loggerReplaceAttrs(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindDuration {
		return slog.String(a.Key, fmt.Sprintf("%dms", a.Value.Duration().Milliseconds()))
	}
	return a
}
