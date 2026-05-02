package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_Fetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mockStatus   int
		mockBody     string
		wantErr      bool
		errContains  string
		validateResp func(t *testing.T, res *results)
	}{
		{
			name:       "success",
			mockStatus: http.StatusOK,
			mockBody: `{
				"status": "ok",
				"totalResults": 100,
				"articles": [
					{
						"title": "Go 1.26 Released",
						"source": { "id": "golang-news", "name": "Go Blog" }
					}
				]
			}`,
			wantErr: false,
			validateResp: func(t *testing.T, res *results) {
				if res.Status != "ok" {
					t.Errorf("expected status 'ok', got %q", res.Status)
				}
				if res.TotalResults != 100 {
					t.Errorf("expected 100 total results, got %d", res.TotalResults)
				}
				if len(res.Articles) != 1 {
					t.Fatalf("expected 1 article, got %d", len(res.Articles))
				}
				if res.Articles[0].Title != "Go 1.26 Released" {
					t.Errorf("unexpected article title: %q", res.Articles[0].Title)
				}
				if res.Articles[0].Source.ID == nil || *res.Articles[0].Source.ID != "golang-news" {
					t.Errorf("unexpected source ID")
				}
			},
		},
		{
			name:       "api error",
			mockStatus: http.StatusUnauthorized,
			mockBody: `{
				"status": "error",
				"code": "apiKeyInvalid",
				"message": "Your API key is invalid or incorrect."
			}`,
			wantErr:     true,
			errContains: "api error (status 401): Your API key is invalid or incorrect.",
		},
		{
			name:        "bad json",
			mockStatus:  http.StatusOK,
			mockBody:    `{ bad json ]`,
			wantErr:     true,
			errContains: "json decoding error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ts := setupMockServer(t, tc.mockStatus, tc.mockBody)
			defer ts.Close()

			client, err := NewClient(Config{
				BaseURL:    ts.URL,
				APIKey:     "test-key",
				PageSize:   10,
				MaxResults: 50,
				Timeout:    1 * time.Second,
			})
			if err != nil {
				t.Fatalf("unexpected error creating client: %v", err)
			}

			res, err := client.Fetch(t.Context(), "golang", 1)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("expected error to contain %q, got %q", tc.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.validateResp != nil {
				tc.validateResp(t, res)
			}
		})
	}
}

// setupMockServer creates an httptest.Server returning the given status code and body.
func setupMockServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/everything" {
			t.Errorf("expected path '/v2/everything', got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "test-key" {
			t.Errorf("expected Authorization header 'test-key', got %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	}))
}
