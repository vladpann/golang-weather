package openweather_transport_http

import (
	"context"
	"net/http"
	"time"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
	"github.com/vladpann/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/golang-weather/internal/features/openweather/service"
)

type OpenWeatherHTTPHandler struct {
	locationsService LocationsService
	config           openweather_service.Config
	authMiddleware   func(http.Handler) http.Handler
	rateLimiter      *ratelimiter.RateLimiter
}

type LocationsService interface {
	CreateLocation(
		ctx context.Context,
		location domain.Location,
	) (domain.Location, error)

	GetLocation(
		ctx context.Context,
		location domain.Location,
	) (domain.Location, error)
}

func NewOpenWeatherHTTPHandler(
	locationsService LocationsService,
	config openweather_service.Config,
	authMiddleware func(http.Handler) http.Handler,
	rateLimiter *ratelimiter.RateLimiter,
) *OpenWeatherHTTPHandler {
	return &OpenWeatherHTTPHandler{
		locationsService: locationsService,
		config:           config,
		authMiddleware:   authMiddleware,
		rateLimiter:      rateLimiter,
	}
}

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func (h *OpenWeatherHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:      http.MethodPost,
			Path:        "/open-weather/get-location",
			Handler:     h.GetOpenWeatherLocation,
			Middlewares: []func(http.Handler) http.Handler{h.authMiddleware},
		},
		{
			Method:  http.MethodPost,
			Path:    "/open-weather/get-weather",
			Handler: h.GetOpenWeather,
		},
	}
}
