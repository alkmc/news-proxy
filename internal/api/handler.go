package api

import (
	"bytes"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type NewsHandler struct {
	client *Client
	tpl    *template.Template
	logger *slog.Logger
}

func NewNewsHandler(client *Client, tpl *template.Template, logger *slog.Logger,
) *NewsHandler {
	return &NewsHandler{
		client: client,
		tpl:    tpl,
		logger: logger,
	}
}

func (h *NewsHandler) Index(w http.ResponseWriter, r *http.Request) {
	h.render(w, nil)
}

func (h *NewsHandler) Search(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	searchKey := params.Get("q")
	page := params.Get("page")
	if page == "" {
		page = "1"
	}

	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "invalid page parameter", http.StatusBadRequest)
		return
	}

	results, err := h.client.Fetch(r.Context(), searchKey, next)
	if err != nil {
		h.logger.Error("failed to fetch news", slog.Any("error", err))
		http.Error(w, "failed to fetch news", http.StatusInternalServerError)
		return
	}

	s := &searchNews{
		SearchKey:   searchKey,
		CurrentPage: next,
		Results:     *results,
		TotalPages:  totalPages(results.TotalResults, h.client.PageSize),
	}

	h.render(w, s)
}

func (h *NewsHandler) render(w http.ResponseWriter, data *searchNews) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	if err := h.tpl.Execute(buf, data); err != nil {
		h.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		h.logger.Error("error writing response", slog.Any("error", err))
	}
}

func totalPages(total, pageSize int) int {
	return int(math.Ceil(float64(total) / float64(pageSize)))
}
