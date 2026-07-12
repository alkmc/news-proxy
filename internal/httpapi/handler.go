package httpapi

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
	"syscall"
	"unicode/utf8"

	"github.com/alkmc/news-proxy/internal/newsapi"
)

const (
	// maxQueryLength caps queries in runes, not bytes.
	maxQueryLength = 200
	// statusClientClosedRequest is nginx's non-standard 499 for a client that disconnected.
	statusClientClosedRequest = 499
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type (
	fetcher interface {
		Fetch(context.Context, string, int) (*newsapi.Results, error)
	}
	// NewsHandler renders the index page and serves search results.
	NewsHandler struct {
		client     fetcher
		tpl        *template.Template
		logger     *slog.Logger
		pageSize   int
		maxResults int
	}
)

// NewNewsHandler builds a NewsHandler with the given client, template, logger, and paging limits.
func NewNewsHandler(client fetcher, tpl *template.Template, logger *slog.Logger,
	pageSize, maxResults int,
) *NewsHandler {
	return &NewsHandler{
		client:     client,
		tpl:        tpl,
		logger:     logger,
		pageSize:   pageSize,
		maxResults: maxResults,
	}
}

// Index renders the empty search page.
func (h *NewsHandler) Index(w http.ResponseWriter, _ *http.Request) {
	h.render(w, http.StatusOK, nil)
}

// Search validates query parameters, fetches articles from NewsAPI, and renders the results page.
func (h *NewsHandler) Search(w http.ResponseWriter, r *http.Request) {
	maxAllowedPages := countPages(h.maxResults, h.pageSize)

	query, page, err := parseSearchParams(r, maxAllowedPages)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.client.Fetch(r.Context(), query, page)
	if err != nil {
		h.handleFetchError(w, err)
		return
	}

	s := &searchPage{
		SearchKey:   query,
		CurrentPage: page,
		Results:     *results,
		TotalPages:  countPages(min(results.TotalResults, h.maxResults), h.pageSize),
	}

	h.render(w, http.StatusOK, s)
}

func (h *NewsHandler) render(w http.ResponseWriter, status int, data *searchPage) {
	buf, ok := bufPool.Get().(*bytes.Buffer)
	if !ok {
		buf = new(bytes.Buffer)
	}
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	if err := h.tpl.Execute(buf, data); err != nil {
		h.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) {
			h.logger.Debug("connection aborted", slog.Any("error", err))
			return
		}
		h.logger.Error("error writing response", slog.Any("error", err))
	}
}

// renderError renders the page with an error message so failures stay styled HTML.
func (h *NewsHandler) renderError(w http.ResponseWriter, status int, msg string) {
	h.render(w, status, &searchPage{Error: msg})
}

func (h *NewsHandler) handleFetchError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) {
		h.logger.Debug("request canceled", slog.Any("error", err))
		w.WriteHeader(statusClientClosedRequest)
		return
	}
	h.logger.Error("failed to fetch news", slog.Any("error", err))

	switch {
	case errors.Is(err, newsapi.ErrUpstreamTimeout):
		h.renderError(w, http.StatusGatewayTimeout, "upstream timeout")
	case errors.Is(err, newsapi.ErrUpstreamRateLimit):
		w.Header().Set("Retry-After", "60")
		h.renderError(w, http.StatusServiceUnavailable, "rate limit exceeded, try later")
	case errors.Is(err, newsapi.ErrUpstreamUnauthorized):
		h.renderError(w, http.StatusBadGateway, "service misconfigured")
	case errors.Is(err, newsapi.ErrUpstreamBadRequest):
		h.renderError(w, http.StatusBadRequest, "invalid search query")
	case errors.Is(err, newsapi.ErrUpstreamServer),
		errors.Is(err, newsapi.ErrUpstreamUnavailable),
		errors.Is(err, newsapi.ErrInvalidResponse):
		h.renderError(w, http.StatusBadGateway, "upstream unavailable")
	default:
		h.renderError(w, http.StatusInternalServerError, "failed to fetch news")
	}
}

func parseSearchParams(r *http.Request, maxAllowedPages int) (query string, page int, err error) {
	q := r.URL.Query()

	query, err = validateQuery(q.Get("q"))
	if err != nil {
		return "", 0, err
	}
	page, err = validatePage(q.Get("page"))
	if err != nil {
		return "", 0, err
	}
	if page > maxAllowedPages {
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

func countPages(total, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}
