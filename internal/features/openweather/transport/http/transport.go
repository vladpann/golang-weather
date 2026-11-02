package openweather_transport_http

import (
	"net/http"

	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	openweather_client "github.com/vladpann/backend/golang-weather/internal/features/openweather/client"
	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
)

type OpenWeatherHTTPHandler struct {
	client         *openweather_client.Client
	authMiddleware core_http_middleware.Middleware
	rateLimiter    *ratelimiter.RateLimiter
}

func NewOpenWeatherHTTPHandler(
	client *openweather_client.Client,
	authMiddleware core_http_middleware.Middleware,
	rateLimiter *ratelimiter.RateLimiter,
) *OpenWeatherHTTPHandler {
	return &OpenWeatherHTTPHandler{
		client:         client,
		authMiddleware: authMiddleware,
		rateLimiter:    rateLimiter,
	}
}

func (h *OpenWeatherHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:      http.MethodGet,
			Path:        "/open-weather/search-location",
			Handler:     h.SearchLocation,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
		{
			Method:      http.MethodGet,
			Path:        "/open-weather/get-weather",
			Handler:     h.GetOpenWeather,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
	}
}
