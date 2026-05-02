package api

import (
	"fmt"
	"time"
)

type source struct {
	ID   *string `json:"id"`
	Name string  `json:"name"`
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
	SearchKey   string
	CurrentPage int
	TotalPages  int
	Results     results
}

func (s *searchNews) IsLastPage() bool {
	return s.CurrentPage >= s.TotalPages
}

func (s *searchNews) NextPage() int {
	return s.CurrentPage + 1
}

func (s *searchNews) PreviousPage() int {
	return s.CurrentPage - 1
}
