package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// maxQueryLength caps queries in runes, not bytes.
const maxQueryLength = 200

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type newsClient interface {
	Fetch(ctx context.Context, searchKey string, page int) (*results, error)
	GetPageSize() int
	GetMaxResults() int
}

type NewsHandler struct {
	client newsClient
	tpl    *template.Template
	logger *slog.Logger
}

func NewNewsHandler(client newsClient, tpl *template.Template, logger *slog.Logger,
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

	searchKey, err := validateQuery(params.Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page, err := validatePage(params.Get("page"))
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
		h.handleFetchError(w, err)
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

func (h *NewsHandler) handleFetchError(w http.ResponseWriter, err error) {
	h.logger.Error("failed to fetch news", slog.Any("error", err))

	switch {
	case errors.Is(err, ErrUpstreamTimeout):
		http.Error(w, "upstream timeout", http.StatusGatewayTimeout)
	case errors.Is(err, ErrUpstreamRateLimit):
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limit exceeded, try later", http.StatusServiceUnavailable)
	case errors.Is(err, ErrUpstreamUnauthorized):
		http.Error(w, "service misconfigured", http.StatusBadGateway)
	case errors.Is(err, ErrUpstreamBadRequest):
		http.Error(w, "invalid search query", http.StatusBadRequest)
	case errors.Is(err, ErrUpstreamServer),
		errors.Is(err, ErrUpstreamUnavailable),
		errors.Is(err, ErrInvalidResponse):
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	default:
		http.Error(w, "failed to fetch news", http.StatusInternalServerError)
	}
}

func validateQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", fmt.Errorf("query is required")
	}
	if utf8.RuneCountInString(q) > maxQueryLength {
		return "", fmt.Errorf("query too long (max %d characters)", maxQueryLength)
	}
	return q, nil
}

func validatePage(pageStr string) (int, error) {
	if pageStr == "" {
		return 1, nil
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, fmt.Errorf("invalid page parameter")
	}
	return page, nil
}

// countPages returns the number of pages, rounding up; 0 if total or pageSize is non-positive.
func countPages(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

// countPagesWithLimit caps total at limit, then counts pages.
func countPagesWithLimit(total, pageSize, limit int) int {
	return countPages(min(total, limit), pageSize)
}
