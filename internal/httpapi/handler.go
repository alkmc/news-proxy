package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/alkmc/news-proxy/internal/newsapi"
	"github.com/alkmc/news-proxy/internal/view"
)

const (
	// maxQueryLength caps queries in runes, not bytes.
	maxQueryLength = 200
	// statusClientClosedRequest is nginx's non-standard 499 for a client that disconnected.
	statusClientClosedRequest = 499
)

type (
	fetcher interface {
		Fetch(context.Context, string, int) (*newsapi.Results, error)
	}
	// Handler renders the index page and serves search results.
	Handler struct {
		client     fetcher
		renderer   *view.Renderer
		logger     *slog.Logger
		pageSize   int
		maxResults int
	}
)

// NewHandler builds a Handler with the given client, renderer, logger, and paging limits.
func NewHandler(
	client fetcher, v *view.Renderer, logger *slog.Logger, pageSize, maxResults int,
) *Handler {
	return &Handler{
		client:     client,
		renderer:   v,
		logger:     logger,
		pageSize:   pageSize,
		maxResults: maxResults,
	}
}

// Index renders the empty search page.
func (h *Handler) Index(w http.ResponseWriter, _ *http.Request) {
	h.renderer.Render(w, http.StatusOK, nil)
}

// ping reports service health for container healthchecks.
func ping(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// Search validates query parameters, fetches articles from NewsAPI, and renders the results page.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	maxAllowedPages := countPages(h.maxResults, h.pageSize)

	query, page, err := parseSearchParams(r, maxAllowedPages)
	if err != nil {
		h.renderer.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	results, err := h.client.Fetch(r.Context(), query, page)
	if err != nil {
		h.handleFetchError(w, err)
		return
	}

	s := &view.SearchPage{
		SearchKey:   query,
		CurrentPage: page,
		Results:     *results,
		TotalPages:  countPages(min(results.TotalResults, h.maxResults), h.pageSize),
	}

	h.renderer.Render(w, http.StatusOK, s)
}

func (h *Handler) handleFetchError(w http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) {
		h.logger.Debug("request canceled", slog.Any("error", err))
		w.WriteHeader(statusClientClosedRequest)
		return
	}
	h.logger.Error("failed to fetch news", slog.Any("error", err))

	switch {
	case errors.Is(err, newsapi.ErrUpstreamTimeout):
		h.renderer.Error(w, http.StatusGatewayTimeout, "upstream timeout")
	case errors.Is(err, newsapi.ErrUpstreamRateLimit):
		w.Header().Set("Retry-After", "60")
		h.renderer.Error(w, http.StatusServiceUnavailable, "rate limit exceeded, try later")
	case errors.Is(err, newsapi.ErrUpstreamUnauthorized):
		h.renderer.Error(w, http.StatusBadGateway, "service misconfigured")
	case errors.Is(err, newsapi.ErrUpstreamBadRequest):
		h.renderer.Error(w, http.StatusBadRequest, "invalid search query")
	case errors.Is(err, newsapi.ErrUpstreamServer),
		errors.Is(err, newsapi.ErrUpstreamUnavailable),
		errors.Is(err, newsapi.ErrInvalidResponse):
		h.renderer.Error(w, http.StatusBadGateway, "upstream unavailable")
	default:
		h.renderer.Error(w, http.StatusInternalServerError, "failed to fetch news")
	}
}

func parseSearchParams(r *http.Request, maxAllowedPages int) (string, int, error) {
	q := r.URL.Query()

	query, err := validateQuery(q.Get("q"))
	if err != nil {
		return "", 0, err
	}
	page, err := validatePage(q.Get("page"))
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
