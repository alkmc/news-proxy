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

// NewsHandler renders the index page and serves search results.
type NewsHandler struct {
	client newsClient
	tpl    *template.Template
	logger *slog.Logger
}

// NewNewsHandler builds a NewsHandler with the given client, template, and logger.
func NewNewsHandler(client newsClient, tpl *template.Template, logger *slog.Logger,
) *NewsHandler {
	return &NewsHandler{
		client: client,
		tpl:    tpl,
		logger: logger,
	}
}

// Index renders the empty search page.
func (h *NewsHandler) Index(w http.ResponseWriter, r *http.Request) {
	h.render(w, nil, isHTMX(r))
}

// Search validates query parameters, fetches articles from NewsAPI, and renders the results page.
func (h *NewsHandler) Search(w http.ResponseWriter, r *http.Request) {
	pageSize := h.client.GetPageSize()
	maxResults := h.client.GetMaxResults()
	maxPages := countPages(maxResults, pageSize)

	query, page, err := parseSearch(r, maxPages)
	s := &searchNews{SearchKey: query}
	withHTMX := isHTMX(r)
	if err != nil {
		s.ErrorMsg = err.Error()
		h.render(w, s, withHTMX)
		return
	}

	results, err := h.client.Fetch(r.Context(), query, page)
	if err != nil {
		h.logger.Error("failed to fetch news", slog.Any("error", err))
		if errors.Is(err, ErrUpstreamRateLimit) {
			w.Header().Set("Retry-After", "60")
		}
		s.ErrorMsg = mapFetchError(err)
		h.render(w, s, withHTMX)
		return
	}

	s.CurrentPage = page
	s.Results = *results
	s.TotalPages = countPages(min(results.TotalResults, maxResults), pageSize)

	h.render(w, s, withHTMX)
}

func (h *NewsHandler) render(w http.ResponseWriter, data *searchNews, withHTMX bool) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	tplName := h.tpl.Name()
	if withHTMX {
		tplName = "results"
	}

	if err := h.tpl.ExecuteTemplate(buf, tplName, data); err != nil {
		h.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		h.logger.Error("error writing response", slog.Any("error", err))
	}
}

func mapFetchError(err error) string {
	switch {
	case errors.Is(err, ErrUpstreamTimeout):
		return "upstream timeout"
	case errors.Is(err, ErrUpstreamRateLimit):
		return "rate limit exceeded, try later"
	case errors.Is(err, ErrUpstreamUnauthorized):
		return "service misconfigured"
	case errors.Is(err, ErrUpstreamBadRequest):
		return "invalid search query"
	case errors.Is(err, ErrUpstreamServer),
		errors.Is(err, ErrUpstreamUnavailable),
		errors.Is(err, ErrInvalidResponse):
		return "upstream unavailable"
	default:
		return "failed to fetch news"
	}
}

func parseSearch(r *http.Request, maxPages int) (query string, page int, err error) {
	q := r.URL.Query()

	query, err = validateQuery(q.Get("q"))
	if err != nil {
		return "", 0, err
	}
	page, err = validatePage(q.Get("page"))
	if err != nil {
		return "", 0, err
	}
	if page > maxPages {
		return "", 0, errors.New("page limit exceeded")
	}
	return query, page, nil
}

func validateQuery(q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", errors.New("query is required")
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
		return 0, errors.New("invalid page parameter")
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

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-History-Restore-Request") != "true"
}
