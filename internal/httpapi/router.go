package httpapi

import (
	"net/http"

	"github.com/alkmc/news-proxy/web"
)

// NewMux builds the application router, configuring routes and applying middlewares.
func NewMux(h *NewsHandler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", staticCache(noDirListing(http.FileServerFS(web.StaticFS))))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)
	return logMD(h.logger)(recoverPanic(securityHeaders(mux)))
}
