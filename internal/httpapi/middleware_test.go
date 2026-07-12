package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	t.Run("CSP", func(t *testing.T) {
		t.Parallel()

		csp := rr.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Fatal("expected Content-Security-Policy header to be set")
		}

		wantDirectives := []string{
			"default-src 'self'",
			"script-src 'self'",
			"frame-ancestors 'none'",
			"base-uri 'none'",
		}
		for _, want := range wantDirectives {
			if !strings.Contains(csp, want) {
				t.Errorf("CSP missing directive %q, got %q", want, csp)
			}
		}
	})

	t.Run("nosniff", func(t *testing.T) {
		t.Parallel()

		if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("expected X-Content-Type-Options 'nosniff', got %q", got)
		}
	})
}
