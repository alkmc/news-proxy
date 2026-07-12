package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alkmc/news-proxy/internal/config"
	"github.com/alkmc/news-proxy/internal/httpapi"
	"github.com/alkmc/news-proxy/internal/newsapi"
	"github.com/alkmc/news-proxy/internal/view"
	"github.com/alkmc/news-proxy/ui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger := config.NewLogger(os.Stdout, cfg.LogLevel)
	slog.SetDefault(logger)
	if err := run(logger, cfg); err != nil {
		logger.Error("proxy failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger, cfg config.Config) error {
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

	var wg sync.WaitGroup
	var shutdownErr error
	wg.Go(func() {
		<-ctx.Done()
		logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			cfg.Server.ShutdownTimeout,
		)
		defer cancel()
		shutdownErr = srv.Shutdown(shutdownCtx)
	})

	logger.Info("server started", slog.String("addr", addr))
	err = srv.ListenAndServe()
	stop()
	wg.Wait()

	if !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server listen failed: %w", err)
	}
	if shutdownErr != nil {
		return fmt.Errorf("server shutdown failed: %w", shutdownErr)
	}
	logger.Info("server shut down gracefully")
	return nil
}
