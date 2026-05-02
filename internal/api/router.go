package api

import (
	"net/http"

	"github.com/alkmc/news-proxy/web"
)

func NewMux(h *NewsHandler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", cacheMiddleware(http.FileServerFS(web.StaticFS)))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)
	return logMD(h.logger)(cspMiddleware(mux))
}
