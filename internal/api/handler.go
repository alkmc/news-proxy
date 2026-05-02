package api

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
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
	client NewsClient
	tpl    *template.Template
	logger *slog.Logger
}

func NewNewsHandler(client NewsClient, tpl *template.Template, logger *slog.Logger,
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

	page, err := h.validatePage(params.Get("page"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pageSize := h.client.GetPageSize()
	maxResults := h.client.GetMaxResults()
	maxAllowedPages := countPages(maxResults, pageSize)

	if page > maxAllowedPages {
		http.Error(w, "page limit exceeded", http.StatusBadRequest)
		return
	}

	results, err := h.client.Fetch(r.Context(), searchKey, page)
	if err != nil {
		h.logger.Error("failed to fetch news", slog.Any("error", err))
		http.Error(w, "failed to fetch news", http.StatusInternalServerError)
		return
	}

	s := &searchNews{
		SearchKey:   searchKey,
		CurrentPage: page,
		Results:     *results,
		TotalPages:  countPagesWithLimit(results.TotalResults, pageSize, maxResults),
	}

	h.render(w, s)
}

func (h *NewsHandler) validatePage(pageStr string) (int, error) {
	if pageStr == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, fmt.Errorf("invalid page parameter")
	}
	return page, nil
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

// countPages calculates the total number of pages based on the total number of items and page size.
func countPages(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

// countPagesWithLimit calculates the total number of pages, capping the total items at the specified limit.
func countPagesWithLimit(total, pageSize, limit int) int {
	return countPages(min(total, limit), pageSize)
}
