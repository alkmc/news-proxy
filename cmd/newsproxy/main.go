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
	"github.com/alkmc/news-proxy/web"
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
	apiKey, err := config.APIKey()
	if err != nil {
		return err
	}
	tpl, err := httpapi.ParseTemplate(web.TemplateFS)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	client, err := newsapi.NewClient(newsapi.Config{
		BaseURL:  config.BaseURL,
		APIKey:   apiKey,
		PageSize: config.PageSize,
		Timeout:  config.FetchTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to create news client: %w", err)
	}

	h := httpapi.NewNewsHandler(client, tpl, logger, config.PageSize, config.MaxResults)
	mux := httpapi.NewMux(h)

	addr := config.ListenAddr()
	srv := httpapi.NewServer(addr, mux)

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
			config.ShutdownTimeout,
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
