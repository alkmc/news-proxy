package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	newsURL = "https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en"
)

type Client struct {
	apiKey     string
	PageSize   int
	httpClient *http.Client
	logger     *slog.Logger
}

func NewClient(apiKey string, pageSize int, logger *slog.Logger) *Client {
	return &Client{
		apiKey:     apiKey,
		PageSize:   pageSize,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (c *Client) Fetch(searchKey string, page int) (*results, error) {
	endpoint := fmt.Sprintf(newsURL, url.QueryEscape(searchKey), c.PageSize, page, c.apiKey)

	var res results
	if err := c.fetch(endpoint, &res); err != nil {
		return nil, err
	}

	return &res, nil
}

func (c *Client) fetch(endpoint string, res *results) error {
	resp, err := c.httpClient.Get(endpoint)
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
