package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alkmc/news-proxy/internal/newsapi"
	"github.com/alkmc/news-proxy/internal/view"
	"github.com/alkmc/news-proxy/ui"
)

func TestRouter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		path         string
		client       *mockNewsClient
		wantStatus   int
		bodyContains string
		wantHeaders  map[string]string
	}{
		{
			name:         "index renders real template",
			path:         "/",
			client:       &mockNewsClient{},
			wantStatus:   http.StatusOK,
			bodyContains: "News Demo",
			wantHeaders: map[string]string{
				"Content-Security-Policy": contentSecurityPolicy,
			},
		},
		{
			name:         "search renders articles",
			path:         "/search?q=golang",
			client:       &mockNewsClient{mockFetchFn: mockFetchResponse(5)},
			wantStatus:   http.StatusOK,
			bodyContains: "Test Article",
		},
		{
			name:         "validation error renders HTML error page",
			path:         "/search",
			client:       &mockNewsClient{},
			wantStatus:   http.StatusBadRequest,
			bodyContains: `<p class="error-message">query is required</p>`,
		},
		{
			name: "fetch error renders HTML error page",
			path: "/search?q=golang",
			client: &mockNewsClient{
				mockFetchFn: func(context.Context, string, int) (*newsapi.Results, error) {
					return nil, newsapi.ErrUpstreamUnavailable
				},
			},
			wantStatus:   http.StatusBadGateway,
			bodyContains: `<p class="error-message">upstream unavailable</p>`,
		},
		{
			name:        "static file served with cache header",
			path:        "/static/style.css",
			client:      &mockNewsClient{},
			wantStatus:  http.StatusOK,
			wantHeaders: map[string]string{"Cache-Control": staticCachePolicy},
		},
		{
			name:       "static directory listing rejected",
			path:       "/static/",
			client:     &mockNewsClient{},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unknown path",
			path:       "/nope",
			client:     &mockNewsClient{},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "handler panic recovered as 500",
			path: "/search?q=golang",
			client: &mockNewsClient{
				mockFetchFn: func(context.Context, string, int) (*newsapi.Results, error) {
					panic("boom")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := newTestServer(t, tc.client)
			status, headers, body := get(t, ts, tc.path)

			if status != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, status)
			}
			if tc.bodyContains != "" && !strings.Contains(body, tc.bodyContains) {
				t.Errorf("expected body to contain %q, got:\n%s", tc.bodyContains, body)
			}
			for key, want := range tc.wantHeaders {
				if got := headers.Get(key); got != want {
					t.Errorf("expected header %s=%q, got %q", key, want, got)
				}
			}
		})
	}
}

// newTestServer starts the full router with the real template and a mock client.
func newTestServer(t *testing.T, client fetcher) *httptest.Server {
	t.Helper()

	tpl, err := view.ParseTemplate(ui.TemplateFS)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.DiscardHandler)
	h := NewHandler(client, view.NewRenderer(tpl, logger), logger, 10, 100)
	ts := httptest.NewServer(NewMux(h))
	t.Cleanup(ts.Close)
	return ts
}

func get(t *testing.T, ts *httptest.Server, path string) (int, http.Header, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, resp.Header, string(body)
}
