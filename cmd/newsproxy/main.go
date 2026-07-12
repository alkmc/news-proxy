package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alkmc/news-proxy/internal/config"
	"github.com/alkmc/news-proxy/internal/httpapi"
	"github.com/alkmc/news-proxy/internal/newsapi"
	"github.com/alkmc/news-proxy/internal/view"
	"github.com/alkmc/news-proxy/ui"
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
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	tpl, err := view.ParseTemplate(ui.TemplateFS)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	client, err := newsapi.NewClient(newsapi.Config{
		BaseURL:  cfg.NewsAPI.BaseURL,
		APIKey:   cfg.NewsAPI.APIKey,
		PageSize: cfg.NewsAPI.PageSize,
		Timeout:  cfg.NewsAPI.FetchTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create news client: %w", err)
	}

	renderer := view.NewRenderer(tpl, logger)
	h := httpapi.NewHandler(client, renderer, logger, cfg.NewsAPI.PageSize, cfg.NewsAPI.MaxResults)
	mux := httpapi.NewMux(h)

	addr := cfg.Server.Address()
	srv := httpapi.NewServer(addr, mux, httpapi.ServerTimeouts{
		Read:       cfg.Server.ReadTimeout,
		ReadHeader: cfg.Server.ReadHeaderTimeout,
		Write:      cfg.Server.WriteTimeout,
		Idle:       cfg.Server.IdleTimeout,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()
	logger.Info("server started", slog.String("addr", addr))

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server listen failed: %w", err)
		}
	case <-ctx.Done():
		logger.Info("signal closing server received")
		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			cfg.Server.ShutdownTimeout,
		)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	logger.Info("server shut down gracefully")
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
