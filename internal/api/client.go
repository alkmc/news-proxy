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

type Client struct {
	baseParsedURL *url.URL
	apiKey        string
	PageSize      int
	httpClient    *http.Client
	logger        *slog.Logger
}

func NewClient(baseURL, apiKey string, pageSize int, logger *slog.Logger) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}
	return &Client{
		baseParsedURL: base,
		apiKey:        apiKey,
		PageSize:      pageSize,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
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

func (c *Client) endpoint(searchKey string, page int) string {
	u := c.baseParsedURL.JoinPath("/v2/everything")

	q := url.Values{
		"q":        {searchKey},
		"pageSize": {strconv.Itoa(c.PageSize)},
		"page":     {strconv.Itoa(page)},
		"apiKey":   {c.apiKey},
		"sortBy":   {"publishedAt"},
		"language": {"en"},
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) fetch(ctx context.Context, endpoint string, res *results) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("could not fetch data: %w", err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var newsErr newsAPIError
		if err := dec.Decode(&newsErr); err != nil {
			c.logger.Error("json decode error",
				slog.Any("error", err),
				slog.Int("status_code", resp.StatusCode),
			)
			return fmt.Errorf("json decoding error (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("news api error (status %d): %s", resp.StatusCode, newsErr.Message)
	}

	if err := dec.Decode(res); err != nil {
		c.logger.Error("json decode error",
			slog.Any("error", err),
			slog.Int("status_code", resp.StatusCode),
		)
		return fmt.Errorf("json decoding error: %w", err)
	}

	return nil
}
