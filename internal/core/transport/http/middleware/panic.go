package core_http_middleware

import (
	"net/http"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

func Panic() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := core_logger.FromContext(ctx)
			responseHandler := core_http_response.NewHTTPResponseHandler(log, w)

			defer func() {
				if p := recover(); p != nil {
					responseHandler.PanicResponse(
						p,
						"during handle HTTP request got unexpected panic",
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
