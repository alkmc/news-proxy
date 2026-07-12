package newsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	httpClient    *http.Client
}

// Config configures the API Client.
type Config struct {
	BaseURL  string
	APIKey   string
	PageSize int
	Timeout  time.Duration
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
		httpClient:    &http.Client{Timeout: cfg.Timeout, Transport: newTransport()},
	}, nil
}

// Fetch returns articles for a query, wrapping upstream failures in sentinel errors.
func (c *Client) Fetch(ctx context.Context, searchKey string, page int) (*Results, error) {
	endpoint := c.endpoint(searchKey, page)

	var res Results
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return &res, nil
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

func (c *Client) fetch(ctx context.Context, endpoint string, res *Results) error {
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

	var apiErr apiError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("%w: status %d: failed to decode body: %w", sentinel, resp.StatusCode, err)
	}
	return fmt.Errorf("%w: status %d: %s", sentinel, resp.StatusCode, apiErr.Message)
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

func newTransport() *http.Transport {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Transport{}
	}
	t := base.Clone()
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 10
	t.MaxConnsPerHost = 10
	t.IdleConnTimeout = 30 * time.Second
	return t
}
