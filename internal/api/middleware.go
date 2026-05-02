package api

import (
	"log/slog"
	"net/http"
	"time"
)

// middleware wraps an http.Handler.
type middleware func(http.Handler) http.Handler

const (
	// staticCachePolicy is the Cache-Control value for static assets (24h).
	staticCachePolicy = "public, max-age=86400"

	// contentSecurityPolicy allows arbitrary HTTPS images for NewsAPI publishers.
	contentSecurityPolicy = "default-src 'self'; script-src 'none'; style-src 'self'; " +
		"img-src 'self' https:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'"
)

func cspMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}

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
