package httpapi

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alkmc/news-proxy/internal/newsapi"
	"github.com/alkmc/news-proxy/internal/view"
)

func TestNewsHandler_Index(t *testing.T) {
	t.Parallel()

	h := setupTestHandler(&mockNewsClient{})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.Index(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "index page") {
		t.Errorf("expected body to contain 'index page', got %q", rr.Body.String())
	}
}

func TestNewsHandler_Search(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		targetURL      string
		mockClient     *mockNewsClient
		expectedStatus int
		bodyContains   string
	}{
		{
			name:      "success with valid query",
			targetURL: "/search?q=golang&page=1",
			mockClient: &mockNewsClient{
				mockFetchFn: mockFetchResponse(5),
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang, Page: 1, TotalPages: 1",
		},
		{
			name:      "success with pagination limit",
			targetURL: "/search?q=golang&page=5",
			mockClient: &mockNewsClient{
				mockFetchFn: mockFetchResponse(150),
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang, Page: 5, TotalPages: 10",
		},
		{
			name:      "empty page defaults to 1",
			targetURL: "/search?q=golang",
			mockClient: &mockNewsClient{
				mockFetchFn: func(_ context.Context, _ string, page int) (*newsapi.Results, error) {
					if page != 1 {
						return nil, errors.New("expected page 1")
					}
					return &newsapi.Results{Status: "ok", TotalResults: 1}, nil
				},
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang, Page: 1, TotalPages: 1",
		},
		{
			name:           "invalid page parameter",
			targetURL:      "/search?q=golang&page=invalid",
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "invalid page parameter",
		},
		{
			name:           "query too long",
			targetURL:      "/search?q=" + url.QueryEscape(strings.Repeat("a", maxQueryLength+1)),
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "query too long",
		},
		{
			name:           "empty query",
			targetURL:      "/search?q=",
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "query is required",
		},
		{
			name:           "missing q parameter",
			targetURL:      "/search",
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "query is required",
		},
		{
			name:           "whitespace-only query",
			targetURL:      "/search?q=" + url.QueryEscape("   "),
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "query is required",
		},
		{
			name:      "query is trimmed before fetch",
			targetURL: "/search?q=" + url.QueryEscape("  golang  "),
			mockClient: &mockNewsClient{
				mockFetchFn: func(_ context.Context, searchKey string, _ int) (*newsapi.Results, error) {
					if searchKey != "golang" {
						return nil, fmt.Errorf("expected trimmed query 'golang', got %q", searchKey)
					}
					return &newsapi.Results{Status: "ok", TotalResults: 0}, nil
				},
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang",
		},
		{
			name:           "query at max length passes validation",
			targetURL:      "/search?q=" + url.QueryEscape(strings.Repeat("a", maxQueryLength)),
			mockClient:     &mockNewsClient{mockFetchFn: mockFetchResponse(0)},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "page exceeds limit",
			targetURL:      "/search?q=golang&page=11",
			mockClient:     &mockNewsClient{},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "page limit exceeded",
		},
		{
			name:      "client fetch error",
			targetURL: "/search?q=golang",
			mockClient: &mockNewsClient{
				mockFetchFn: func(context.Context, string, int) (*newsapi.Results, error) {
					return nil, errors.New("upstream timeout")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			bodyContains:   "failed to fetch news",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := setupTestHandler(tc.mockClient)
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, tc.targetURL, nil)
			rr := httptest.NewRecorder()

			h.Search(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d", tc.expectedStatus, rr.Code)
			}
			if tc.bodyContains != "" && !strings.Contains(rr.Body.String(), tc.bodyContains) {
				t.Errorf("expected body to contain %q, got %q", tc.bodyContains, rr.Body.String())
			}
		})
	}
}

// setupTestHandler configures a NewsHandler with a dummy template, silent logger, and 10/100 paging.
func setupTestHandler(client fetcher) *NewsHandler {
	tplStr := `{{if .}}{{if .Error}}Error: {{.Error}}{{else}}Query: {{.SearchKey}}, ` +
		`Page: {{.CurrentPage}}, TotalPages: {{.TotalPages}}{{end}}{{else}}index page{{end}}`
	tpl := template.Must(template.New("index.html").Parse(tplStr))
	logger := slog.New(slog.DiscardHandler)
	return NewNewsHandler(client, view.NewRenderer(tpl, logger), logger, 10, 100)
}

func mockFetchResponse(totalResults int) func(context.Context, string, int) (*newsapi.Results, error) {
	return func(_ context.Context, _ string, _ int) (*newsapi.Results, error) {
		return &newsapi.Results{
			Status:       "ok",
			TotalResults: totalResults,
			Articles:     []newsapi.Article{{Title: "Test Article"}},
		}, nil
	}
}

// mockNewsClient implements the fetcher interface for testing.
type mockNewsClient struct {
	mockFetchFn func(context.Context, string, int) (*newsapi.Results, error)
}

func (m *mockNewsClient) Fetch(ctx context.Context, searchKey string, page int) (*newsapi.Results, error) {
	if m.mockFetchFn != nil {
		return m.mockFetchFn(ctx, searchKey, page)
	}
	return &newsapi.Results{}, nil
}
