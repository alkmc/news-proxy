package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	endpointTpl = "%s/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en"
)

type Client struct {
	baseURL    string
	apiKey     string
	PageSize   int
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(baseURL, apiKey string, pageSize int, logger *slog.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		PageSize:   pageSize,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (c *Client) Fetch(ctx context.Context, searchKey string, page int) (*results, error) {
	endpoint := c.endpoint(searchKey, page)

	var res results
	if err := c.fetch(ctx, endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) endpoint(searchKey string, page int) string {
	return fmt.Sprintf(endpointTpl, c.baseURL, url.QueryEscape(searchKey), c.PageSize, page, c.apiKey)
}

func (c *Client) fetch(ctx context.Context, endpoint string, res *results) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return errors.New("could not create request")
	}

	resp, err := c.httpClient.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return errors.New("could not fetch data")
	}

	dec := json.NewDecoder(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var newsErr newsAPIError
		if err := dec.Decode(&newsErr); err != nil {
			c.logger.Error("json decode error", slog.Any("error", err))
			return errors.New("json decoding error")
		}
		return errors.New(newsErr.Message)
	}

	if err := dec.Decode(res); err != nil {
		c.logger.Error("json decode error", slog.Any("error", err))
		return errors.New("json decoding error")
	}

	return nil
}
