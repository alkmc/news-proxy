package httpapi

import "github.com/alkmc/news-proxy/internal/newsapi"

// searchPage is the template view model for a search results page.
type searchPage struct {
	SearchKey   string
	CurrentPage int
	TotalPages  int
	// Error, when set, renders instead of results.
	Error   string
	Results newsapi.Results
}

func (s *searchPage) IsLastPage() bool {
	return s.CurrentPage >= s.TotalPages
}

func (s *searchPage) NextPage() int {
	return s.CurrentPage + 1
}

func (s *searchPage) PreviousPage() int {
	return s.CurrentPage - 1
}
