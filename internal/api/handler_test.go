package api

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewsHandler_Index(t *testing.T) {
	t.Parallel()

	h := setupTestHandler(&mockNewsClient{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
				pageSize:    10,
				maxResults:  100,
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang, Page: 1, TotalPages: 1",
		},
		{
			name:      "success with pagination limit",
			targetURL: "/search?q=golang&page=5",
			mockClient: &mockNewsClient{
				mockFetchFn: mockFetchResponse(150),
				pageSize:    10,
				maxResults:  100,
			},
			expectedStatus: http.StatusOK,
			bodyContains:   "Query: golang, Page: 5, TotalPages: 10",
		},
		{
			name:      "empty page defaults to 1",
			targetURL: "/search?q=golang",
			mockClient: &mockNewsClient{
				mockFetchFn: func(ctx context.Context, searchKey string, page int) (*results, error) {
					if page != 1 {
						return nil, errors.New("expected page 1")
					}
					return &results{Status: "ok", TotalResults: 1}, nil
				},
				pageSize:   10,
				maxResults: 100,
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
			name:      "page exceeds limit",
			targetURL: "/search?q=golang&page=11",
			mockClient: &mockNewsClient{
				pageSize:   10,
				maxResults: 100,
			},
			expectedStatus: http.StatusBadRequest,
			bodyContains:   "page limit exceeded",
		},
		{
			name:      "client fetch error",
			targetURL: "/search?q=golang",
			mockClient: &mockNewsClient{
				mockFetchFn: func(context.Context, string, int) (*results, error) {
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

// setupTestHandler configures a NewsHandler with a dummy template and silent logger.
func setupTestHandler(client newsClient) *NewsHandler {
	tplStr := `{{if .}}Query: {{.SearchKey}}, Page: {{.CurrentPage}}, TotalPages: {{.TotalPages}}{{else}}index page{{end}}`
	tpl := template.Must(template.New("index.html").Parse(tplStr))
	logger := slog.New(slog.DiscardHandler)
	return NewNewsHandler(client, tpl, logger)
}

func mockFetchResponse(totalResults int) func(context.Context, string, int) (*results, error) {
	return func(ctx context.Context, searchKey string, page int) (*results, error) {
		return &results{
			Status:       "ok",
			TotalResults: totalResults,
			Articles:     []article{{Title: "Test Article"}},
		}, nil
	}
}

// mockNewsClient implements the newsClient interface for testing.
type mockNewsClient struct {
	mockFetchFn func(context.Context, string, int) (*results, error)
	pageSize    int
	maxResults  int
}

func (m *mockNewsClient) Fetch(ctx context.Context, searchKey string, page int) (*results, error) {
	if m.mockFetchFn != nil {
		return m.mockFetchFn(ctx, searchKey, page)
	}
	return &results{}, nil
}

func (m *mockNewsClient) GetPageSize() int {
	if m.pageSize == 0 {
		return 10
	}
	return m.pageSize
}

func (m *mockNewsClient) GetMaxResults() int {
	if m.maxResults == 0 {
		return 100
	}
	return m.maxResults
}
