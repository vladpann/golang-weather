package core_http_middleware

import (
	"net/http"
	"time"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
	"go.uber.org/zap"
)

func Trace() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := core_logger.FromContext(ctx)
			rw := core_http_response.NewResponseWriter(w)

			before := time.Now()
			log.Debug(
				">>> incoming HTTP request",
				zap.String("http method", r.Method),
				zap.Time("time", before.UTC()),
			)

			next.ServeHTTP(rw, r)

			log.Debug(
				"<<< done HTTP request",
				zap.Int("status_code", rw.GetStatusCode()),
				zap.Duration("latency", time.Since(before)),
			)
		})
	}
}
