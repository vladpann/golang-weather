package openweather_client

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrInvalidAPIKey indicates that the API key is invalid or missing
	ErrInvalidAPIKey = errors.New("invalid or missing API key")

	// ErrRateLimitExceeded indicates that the upstream rate limit was exceeded
	ErrRateLimitExceeded = errors.New("upstream rate limit exceeded")

	// ErrServiceUnavailable indicates that the upstream service is temporarily unavailable
	ErrServiceUnavailable = errors.New("upstream service unavailable")

	// ErrBadGateway indicates that the upstream returned an invalid response
	ErrBadGateway = errors.New("invalid response from upstream")

	// ErrGatewayTimeout indicates that the upstream request timed out
	ErrGatewayTimeout = errors.New("upstream request timeout")

	// ErrNotFound indicates that the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrBadRequest indicates that the request parameters are invalid
	ErrBadRequest = errors.New("bad request")

	// ErrBodyTooLarge indicates that the response body exceeded the size limit
	ErrBodyTooLarge = errors.New("response body too large")
)

// UpstreamError wraps upstream errors with context
type UpstreamError struct {
	StatusCode   int
	Message      string
	RetryAfter   int // seconds, if Retry-After header present
	Err          error
	IsRetryable  bool
	IsClientAuth bool // true if this is an API key problem, not client authentication
}

func (e *UpstreamError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("upstream error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("upstream error (status %d): %v", e.StatusCode, e.Err)
}

func (e *UpstreamError) Unwrap() error {
	return e.Err
}

// HTTPStatusCode maps the error to an appropriate HTTP status code for client responses
func (e *UpstreamError) HTTPStatusCode() int {
	switch {
	case errors.Is(e.Err, ErrInvalidAPIKey):
		// Don't expose upstream auth issues as client auth issues
		return http.StatusBadGateway
	case errors.Is(e.Err, ErrRateLimitExceeded):
		return http.StatusServiceUnavailable
	case errors.Is(e.Err, ErrServiceUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(e.Err, ErrBadGateway):
		return http.StatusBadGateway
	case errors.Is(e.Err, ErrGatewayTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(e.Err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(e.Err, ErrBadRequest):
		return http.StatusBadRequest
	case errors.Is(e.Err, ErrBodyTooLarge):
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}
