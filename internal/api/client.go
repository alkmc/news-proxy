package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type NewsClient interface {
	Fetch(ctx context.Context, searchKey string, page int) (*results, error)
	GetPageSize() int
	GetMaxResults() int
}

type Client struct {
	baseParsedURL *url.URL
	apiKey        string
	pageSize      int
	maxResults    int
	httpClient    *http.Client
	logger        *slog.Logger
}

// Config defines the configuration for the API Client.
type Config struct {
	BaseURL    string
	APIKey     string
	PageSize   int
	MaxResults int
	Timeout    time.Duration
	Logger     *slog.Logger
}

func NewClient(cfg Config) (*Client, error) {
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		baseParsedURL: base,
		apiKey:        cfg.APIKey,
		pageSize:      cfg.PageSize,
		maxResults:    cfg.MaxResults,
		httpClient:    &http.Client{Timeout: cfg.Timeout},
		logger:        logger,
	}, nil
}

func (c *Client) Fetch(ctx context.Context, searchKey string, page int) (*results, error) {
	endpoint := c.endpoint(searchKey, page)

	var res results
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return &res, nil
}

func (c *Client) GetPageSize() int {
	return c.pageSize
}

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
		return fmt.Errorf("could not fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var newsErr newsAPIError
		if err := json.NewDecoder(resp.Body).Decode(&newsErr); err != nil {
			return fmt.Errorf("json decoding error (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, newsErr.Message)
	}

	if err := json.NewDecoder(resp.Body).Decode(res); err != nil {
		return fmt.Errorf("json decoding error: %w", err)
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	return req, nil
}
