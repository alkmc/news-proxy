package api

import (
	"net/http"

	"github.com/alkmc/news-proxy/web"
)

func NewMux(h *NewsHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", CacheMiddleware(http.FileServerFS(web.FS)))
	mux.HandleFunc("GET /search", h.Search)
	mux.HandleFunc("GET /{$}", h.Index)
	return mux
}
