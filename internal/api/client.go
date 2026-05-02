package api

import (
	"context"
	"encoding/json"
	"errors"
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
		return nil, err
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
		return nil, err
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
