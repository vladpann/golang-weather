package openweather_transport_http

import (
	"net/http"
	"time"

	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/backend/golang-weather/internal/features/openweather/service"
)

type OpenWeatherHTTPHandler struct {
	config         openweather_service.Config
	authMiddleware core_http_middleware.Middleware
	rateLimiter    *ratelimiter.RateLimiter
}

func NewOpenWeatherHTTPHandler(
	config openweather_service.Config,
	authMiddleware core_http_middleware.Middleware,
	rateLimiter *ratelimiter.RateLimiter,
) *OpenWeatherHTTPHandler {
	return &OpenWeatherHTTPHandler{
		config:         config,
		authMiddleware: authMiddleware,
		rateLimiter:    rateLimiter,
	}
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
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
