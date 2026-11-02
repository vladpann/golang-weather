package core_http_middleware

import (
	"net/http"
	"sync/atomic"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

// LoadShedConfig configures load shedding behavior
type LoadShedConfig struct {
	// MaxConcurrentRequests is the maximum number of requests being processed concurrently.
	// When this limit is reached, new requests receive 503 Service Unavailable.
	// Set to 0 to disable load shedding.
	MaxConcurrentRequests int64
}

// LoadShed creates a middleware that sheds load by rejecting new requests when
// the server is at capacity. This provides controlled degradation under heavy load
// instead of accumulating requests until the server crashes.
//
// Returns 503 Service Unavailable with Retry-After header when capacity is reached.
func LoadShed(config LoadShedConfig, logger *core_logger.Logger) Middleware {
	if config.MaxConcurrentRequests <= 0 {
		// Load shedding disabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	var activeRequests int64

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to increment active request counter
			current := atomic.AddInt64(&activeRequests, 1)
			
			if current > config.MaxConcurrentRequests {
				// Over capacity - shed this request
				atomic.AddInt64(&activeRequests, -1)
				
				logger.Warn("load shedding: rejecting request",
					zap.Int64("active_requests", current-1),
					zap.Int64("max_concurrent", config.MaxConcurrentRequests),
					zap.String("path", r.URL.Path),
				)
				
				w.Header().Set("Retry-After", "5")
				http.Error(w, "Service temporarily overloaded, please retry", http.StatusServiceUnavailable)
				return
			}

			// Process the request
			defer func() {
				atomic.AddInt64(&activeRequests, -1)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
