package httpapi

import (
	"net/http"

	"github.com/alkmc/news-proxy/ui"
)

// NewMux builds the application router, configuring routes and applying middlewares.
func NewMux(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", staticCache(noDirListing(http.FileServerFS(ui.StaticFS))))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)
	return logMD(h.logger)(recoverPanic(securityHeaders(mux)))
}
