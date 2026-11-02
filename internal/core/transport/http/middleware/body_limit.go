package core_http_middleware

import (
	"net/http"
)

// MaxBodySize creates a middleware that limits the maximum size of request bodies.
// This must be applied BEFORE any middleware that reads the body (like Logger).
// It prevents memory exhaustion from maliciously large request bodies.
func MaxBodySize(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if maxBytes > 0 && r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
