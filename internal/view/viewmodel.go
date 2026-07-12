package view

import "github.com/alkmc/news-proxy/internal/newsapi"

// SearchPage is the template view model for a search results page.
type SearchPage struct {
	SearchKey   string
	CurrentPage int
	TotalPages  int
	// Error, when set, renders instead of results.
	Error   string
	Results newsapi.Results
}

func (s *SearchPage) IsLastPage() bool {
	return s.CurrentPage >= s.TotalPages
}

func (s *SearchPage) NextPage() int {
	return s.CurrentPage + 1
}

func (s *SearchPage) PreviousPage() int {
	return s.CurrentPage - 1
}
