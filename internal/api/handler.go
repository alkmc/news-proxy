package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type NewsHandler struct {
	apiKey     string
	tpl        *template.Template
	httpClient *http.Client
	logger     *slog.Logger
}

func NewNewsHandler(apiKey string, tpl *template.Template, logger *slog.Logger) *NewsHandler {
	return &NewsHandler{
		apiKey:     apiKey,
		tpl:        tpl,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (h *NewsHandler) Index(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := h.tpl.Execute(&buf, nil); err != nil {
		h.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		h.logger.Error("error writing response", slog.Any("error", err))
	}
}

func (h *NewsHandler) Search(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	params := u.Query()
	searchKey := params.Get("q")
	page := params.Get("page")
	if page == "" {
		page = "1"
	}

	s := &searchNews{}
	s.SearchKey = searchKey

	next, err := strconv.Atoi(page)
	if err != nil {
		http.Error(w, "unexpected server error", http.StatusInternalServerError)
		return
	}
	s.CurrentPage = next

	const (
		URL      = "https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en"
		pageSize = 20
	)

	endpoint := fmt.Sprintf(URL, url.QueryEscape(s.SearchKey), pageSize, s.CurrentPage, h.apiKey)
	if err := h.fetch(endpoint, &s.Results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.TotalPages = totalPages(s.Results.TotalResults, pageSize)

	var buf bytes.Buffer
	if err := h.tpl.Execute(&buf, s); err != nil {
		h.logger.Error("template execution error", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		h.logger.Error("error writing response", slog.Any("error", err))
	}
}

func (h *NewsHandler) fetch(endpoint string, v any) error {
	resp, err := h.httpClient.Get(endpoint)
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
			h.logger.Error("json decode error", slog.Any("error", err))
			return errors.New("json decoding error")
		}
		return errors.New(newsErr.Message)
	}

	if err := dec.Decode(v); err != nil {
		h.logger.Error("json decode error", slog.Any("error", err))
		return errors.New("json decoding error")
	}

	return nil
}

func totalPages(total, pageSize int) int {
	return int(math.Ceil(float64(total) / float64(pageSize)))
}
