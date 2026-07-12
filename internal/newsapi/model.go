package newsapi

import "time"

type (
	// Source identifies the publisher of an article.
	Source struct {
		ID   *string `json:"id"`
		Name string  `json:"name"`
	}
	// Article is a single NewsAPI search result.
	Article struct {
		Source      Source    `json:"source"`
		Author      string    `json:"author"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		URL         string    `json:"url"`
		URLToImage  string    `json:"urlToImage"`
		PublishedAt time.Time `json:"publishedAt"`
		Content     string    `json:"content"`
	}
	// Results is the NewsAPI /v2/everything response.
	Results struct {
		Status       string    `json:"status"`
		TotalResults int       `json:"totalResults"`
		Articles     []Article `json:"articles"`
	}
	apiError struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
)
