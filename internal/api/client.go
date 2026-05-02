package api

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client calls the NewsAPI /v2/everything endpoint with bounded paging.
type Client struct {
	baseParsedURL *url.URL
	apiKey        string
	pageSize      int
	maxResults    int
	httpClient    *http.Client
	logger        *slog.Logger
}

// Config configures the API Client.
type Config struct {
	BaseURL    string
	APIKey     string
	PageSize   int
	MaxResults int
	Timeout    time.Duration
	Logger     *slog.Logger
}

// NewClient parses the base URL and returns a configured Client.
func NewClient(cfg Config) (*Client, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	return &Client{
		baseParsedURL: base,
		apiKey:        cfg.APIKey,
		pageSize:      cfg.PageSize,
		maxResults:    cfg.MaxResults,
		httpClient:    &http.Client{Timeout: cfg.Timeout, Transport: customTransport()},
		logger:        cmp.Or(cfg.Logger, slog.Default()),
	}, nil
}

// Fetch returns articles for a query, wrapping upstream failures in sentinel errors.
func (c *Client) Fetch(ctx context.Context, searchKey string, page int) (*results, error) {
	endpoint := c.endpoint(searchKey, page)

	var res results
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return &res, nil
}

// GetPageSize returns the configured page size.
func (c *Client) GetPageSize() int {
	return c.pageSize
}

// GetMaxResults returns the configured cap on total results.
func (c *Client) GetMaxResults() int {
	return c.maxResults
}

func (c *Client) endpoint(searchKey string, page int) string {
	u := c.baseParsedURL.JoinPath("/v2/everything")

	q := url.Values{
		"q":        {searchKey},
		"pageSize": {strconv.Itoa(c.pageSize)},
		"page":     {strconv.Itoa(page)},
		"sortBy":   {"publishedAt"},
		"language": {"en"},
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) fetch(ctx context.Context, endpoint string, res *results) error {
	req, err := c.newRequest(ctx, endpoint)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return classifyTransportError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeUpstreamError(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return nil
}

// classifyTransportError maps timeouts and network errors to sentinel errors.
func classifyTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %w", ErrUpstreamTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrUpstreamUnavailable, err)
}

// decodeUpstreamError wraps a non-2xx response in a status-specific sentinel.
func decodeUpstreamError(resp *http.Response) error {
	sentinel := classifyStatus(resp.StatusCode)

	var newsErr newsAPIError
	if err := json.NewDecoder(resp.Body).Decode(&newsErr); err != nil {
		return fmt.Errorf("%w: status %d: failed to decode body: %w", sentinel, resp.StatusCode, err)
	}
	return fmt.Errorf("%w: status %d: %s", sentinel, resp.StatusCode, newsErr.Message)
}

func classifyStatus(status int) error {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return ErrUpstreamUnauthorized
	case status == http.StatusTooManyRequests:
		return ErrUpstreamRateLimit
	case status == http.StatusBadRequest:
		return ErrUpstreamBadRequest
	case status >= 500:
		return ErrUpstreamServer
	default:
		return ErrUpstreamServer
	}
}

func (c *Client) newRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	return req, nil
}

func customTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 10
	transport.MaxConnsPerHost = 10
	transport.IdleConnTimeout = 30 * time.Second
	return transport
}
