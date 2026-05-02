package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSPMiddleware(t *testing.T) {
	t.Parallel()

	handler := cspMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	got := rr.Header().Get("Content-Security-Policy")
	if got == "" {
		t.Fatal("expected Content-Security-Policy header to be set")
	}

	wantDirectives := []string{
		"default-src 'self'",
		"script-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'none'",
	}
	for _, want := range wantDirectives {
		if !strings.Contains(got, want) {
			t.Errorf("CSP missing directive %q, got %q", want, got)
		}
	}
}
