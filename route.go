package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type newsHandler struct {
	apiKey     string
	tpl        *template.Template
	httpClient *http.Client
}

func newNewsHandler(apiKey string, tpl *template.Template) *newsHandler {
	return &newsHandler{
		apiKey:     apiKey,
		tpl:        tpl,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *newsHandler) index(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := h.tpl.Execute(&buf, nil); err != nil {
		log.Printf("template execution error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("error writing response: %v", err)
	}
}

func (h *newsHandler) search(w http.ResponseWriter, r *http.Request) {
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
	s.NextPage = next

	const (
		URL      = "https://newsapi.org/v2/everything?q=%s&pageSize=%d&page=%d&apiKey=%s&sortBy=publishedAt&language=en"
		pageSize = 20
	)

	endpoint := fmt.Sprintf(URL, url.QueryEscape(s.SearchKey), pageSize, s.NextPage, h.apiKey)
	if err := h.fetch(endpoint, &s.Results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.TotalPages = totalPages(s.Results.TotalResults, pageSize)

	if ok := !s.IsLastPage(); ok {
		s.NextPage++
	}

	var buf bytes.Buffer
	if err := h.tpl.Execute(&buf, s); err != nil {
		log.Printf("template execution error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("error writing response: %v", err)
	}
}

func (h *newsHandler) fetch(endpoint string, v any) error {
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
			log.Println(err.Error())
			return errors.New("json decoding error")
		}
		return errors.New(newsErr.Message)
	}

	if err := dec.Decode(v); err != nil {
		log.Println(err.Error())
		return errors.New("json decoding error")
	}

	return nil
}

func totalPages(total, pageSize int) int {
	return int(math.Ceil(float64(total) / float64(pageSize)))
}
