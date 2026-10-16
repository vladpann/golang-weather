package openweather_transport_http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	core_http_response "github.com/vladpann/golang-weather/internal/core/transport/http/response"
	"github.com/vladpann/golang-weather/internal/features/openweather/ratelimiter"
)

// handleRateLimitError converts rate limiter errors to appropriate HTTP responses
func (h *OpenWeatherHTTPHandler) handleRateLimitError(
	responseHandler *core_http_response.HTTPResponseHandler,
	err error,
) {
	switch {
	case errors.Is(err, ratelimiter.ErrQueueFull):
		responseHandler.JSONResponse(
			map[string]interface{}{
				"error": "Service temporarily unavailable",
				"message": fmt.Sprintf(
					"Rate limit queue is full (max queue size: %d). Please try again in a few seconds.",
					h.config.QueueSize,
				),
				"rate_limit": map[string]interface{}{
					"requests_per_minute": h.config.RequestsPerMinute,
					"retry_after_seconds": 5,
				},
			},
			http.StatusServiceUnavailable,
		)

	case errors.Is(err, context.DeadlineExceeded):
		responseHandler.JSONResponse(
			map[string]interface{}{
				"error":   "Request timeout",
				"message": "Request timed out while waiting for rate limit. Please try again.",
				"rate_limit": map[string]interface{}{
					"requests_per_minute": h.config.RequestsPerMinute,
					"retry_after_seconds": 5,
				},
			},
			http.StatusRequestTimeout,
		)

	case errors.Is(err, context.Canceled):
		responseHandler.JSONResponse(
			map[string]interface{}{
				"error":   "Request canceled",
				"message": "Request was canceled",
			},
			http.StatusRequestTimeout,
		)

	default:
		// Generic rate limit error
		responseHandler.JSONResponse(
			map[string]interface{}{
				"error": "Rate limit exceeded",
				"message": fmt.Sprintf(
					"Too many requests. Rate limit: %d requests per minute. Please try again in a few seconds.",
					h.config.RequestsPerMinute,
				),
				"rate_limit": map[string]interface{}{
					"requests_per_minute": h.config.RequestsPerMinute,
					"retry_after_seconds": 5,
				},
			},
			http.StatusTooManyRequests,
		)
	}
}
