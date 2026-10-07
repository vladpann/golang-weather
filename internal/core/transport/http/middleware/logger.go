package core_http_middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

type Middleware func(http.Handler) http.Handler

const requestIDHeader = "X-Request-ID"

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			r.Header.Set(requestIDHeader, requestID)
			w.Header().Set(requestIDHeader, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func HTTPLogger(logger *core_logger.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := r.Header.Get(requestIDHeader)

			l := logger.With(
				zap.String("request_id", requestID),
				zap.String("url", r.URL.String()),
			)

			ctx := core_logger.ToContext(r.Context(), l)

			var requestBody string
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					requestBody = string(bodyBytes)
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			wrappedWriter := newResponseWriter(w)

			next.ServeHTTP(wrappedWriter, r.WithContext(ctx))

			duration := time.Since(start)
			durationMs := float64(duration.Microseconds()) / 1000.0

			reqField := r.Method + " " + r.RequestURI
			if requestBody != "" {
				reqField += " Body: " + requestBody
			}

			l.Info("HTTP Request",
				zap.String("time", start.Format(time.RFC3339)),
				zap.String("req", reqField),
				zap.String("resp", wrappedWriter.body.String()),
				zap.Int("code", wrappedWriter.statusCode),
				zap.Float64("duration_ms", durationMs),
			)
		})
	}
}
