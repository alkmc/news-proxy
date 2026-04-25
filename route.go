package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.Execute(w, nil)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
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

	endpoint := fmt.Sprintf(URL, url.QueryEscape(s.SearchKey), pageSize, s.NextPage, *apiKey)
	if err := fetch(endpoint, &s.Results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	s.TotalPages = totalPages(s.Results.TotalResults, pageSize)

	if ok := !s.IsLastPage(); ok {
		s.NextPage++
	}
	if err := tpl.Execute(w, s); err != nil {
		log.Fatal(err)
	}
}

func fetch(endpoint string, v any) error {
	resp, err := http.Get(endpoint)
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
