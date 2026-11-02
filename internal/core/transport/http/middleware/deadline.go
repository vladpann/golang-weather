package core_http_middleware

import (
	"context"
	"net/http"
	"time"
)

// RequestDeadline creates a middleware that enforces a timeout for the entire request lifecycle,
// covering database queries, rate limiter queues, and upstream API calls.
// This prevents requests from accumulating indefinitely when downstream services are slow.
func RequestDeadline(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip deadline if timeout is 0 or negative
			if timeout <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Create context with deadline for the entire request
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Replace request context
			r = r.WithContext(ctx)

			// Handle the request
			next.ServeHTTP(w, r)
		})
	}
}
