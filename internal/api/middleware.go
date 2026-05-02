package api

import (
	"log/slog"
	"net/http"
	"time"
)

// middleware is a function that wraps an http.Handler.
type middleware func(http.Handler) http.Handler

// staticCachePolicy defines a 24-hour cache policy (in seconds) for static assets.
const staticCachePolicy = "public, max-age=86400"

func cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", staticCachePolicy)
		next.ServeHTTP(w, r)
	})
}

// logMD logs method, path, and request duration.
func logMD(logger *slog.Logger) middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				logger.Info("http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Duration("duration", time.Since(start)),
				)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
