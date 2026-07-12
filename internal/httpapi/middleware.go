package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	// staticCachePolicy is the Cache-Control value for static assets (24h).
	staticCachePolicy = "public, max-age=86400"

	// contentSecurityPolicy allows arbitrary external images.
	contentSecurityPolicy = "default-src 'self'; script-src 'self'; img-src 'self' https: http:; " +
		"form-action 'self'; frame-ancestors 'none'; base-uri 'none'"
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

// noDirListing rejects directory paths so the file server never lists contents.
func noDirListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func staticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", staticCachePolicy)
		next.ServeHTTP(w, r)
	})
}

// logMD logs request metadata: method, path, and duration.
func logMD(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				logger.Info(
					"http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Duration("duration", time.Since(start)),
				)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
