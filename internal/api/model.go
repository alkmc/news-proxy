package api

import (
	"fmt"
	"time"
)

type source struct {
	ID   any    `json:"id"`
	Name string `json:"name"`
}

type article struct {
	Source      source    `json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	URLToImage  string    `json:"urlToImage"`
	PublishedAt time.Time `json:"publishedAt"`
	Content     string    `json:"content"`
}

func (a *article) FormatPublishedDate() string {
	year, month, day := a.PublishedAt.Date()
	return fmt.Sprintf("%v %d, %d", month, day, year)
}

type results struct {
	Status       string    `json:"status"`
	TotalResults int       `json:"totalResults"`
	Articles     []article `json:"articles"`
}

type newsAPIError struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type searchNews struct {
	SearchKey  string
	NextPage   int
	TotalPages int
	Results    results
}

func (s *searchNews) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *searchNews) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *searchNews) PreviousPage() int {
	return s.CurrentPage() - 1
}
