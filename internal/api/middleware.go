package api

import "net/http"

// staticCachePolicy defines a 24-hour cache policy (in seconds) for static assets.
const staticCachePolicy = "public, max-age=86400"

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", staticCachePolicy)
		next.ServeHTTP(w, r)
	})
}
