package core_http_middleware

import (
	"net/http"
	"time"

	core_metrics "github.com/vladpann/backend/golang-weather/internal/core/metrics"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

// Metrics creates middleware that records HTTP request metrics
func Metrics(m *core_metrics.Metrics) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track in-flight requests
			m.HTTPRequestsInFlight.Inc()
			defer m.HTTPRequestsInFlight.Dec()

			// Wrap response writer to capture status code
			rw := core_http_response.NewResponseWriter(w)

			start := time.Now()
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()

			// Record metrics
			statusCode := rw.GetStatusCode()
			method := r.Method
			statusClass := core_metrics.StatusClass(statusCode)

			m.HTTPRequestsTotal.WithLabelValues(method, statusClass).Inc()
			m.HTTPRequestDuration.WithLabelValues(method, statusClass).Observe(duration)
		})
	}
}
