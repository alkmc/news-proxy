package api

import "errors"

var (
	ErrUpstreamUnavailable  = errors.New("upstream unavailable")
	ErrUpstreamTimeout      = errors.New("upstream timeout")
	ErrUpstreamRateLimit    = errors.New("upstream rate limit exceeded")
	ErrUpstreamUnauthorized = errors.New("upstream unauthorized")
	ErrUpstreamBadRequest   = errors.New("upstream rejected request")
	ErrUpstreamServer       = errors.New("upstream server error")
	ErrInvalidResponse      = errors.New("invalid upstream response")
)
