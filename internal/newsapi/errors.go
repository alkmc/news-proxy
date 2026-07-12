package newsapi

import "errors"

var (
	// ErrUpstreamUnavailable signals a transport-level failure reaching NewsAPI.
	ErrUpstreamUnavailable = errors.New("upstream unavailable")
	// ErrUpstreamTimeout signals a context deadline or cancellation reaching NewsAPI.
	ErrUpstreamTimeout = errors.New("upstream timeout")
	// ErrUpstreamRateLimit signals NewsAPI returned HTTP 429.
	ErrUpstreamRateLimit = errors.New("upstream rate limit exceeded")
	// ErrUpstreamUnauthorized signals NewsAPI returned 401 or 403 (likely bad API key).
	ErrUpstreamUnauthorized = errors.New("upstream unauthorized")
	// ErrUpstreamBadRequest signals NewsAPI returned HTTP 400.
	ErrUpstreamBadRequest = errors.New("upstream rejected request")
	// ErrUpstreamServer signals NewsAPI returned a 5xx response.
	ErrUpstreamServer = errors.New("upstream server error")
	// ErrInvalidResponse signals NewsAPI returned a body that failed to decode.
	ErrInvalidResponse = errors.New("invalid upstream response")
)
