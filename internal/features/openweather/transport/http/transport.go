package openweather_transport_http

import (
	"context"
	"net/http"
	"time"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/backend/golang-weather/internal/features/openweather/service"
)

type OpenWeatherHTTPHandler struct {
	locationsService LocationsService
	config           openweather_service.Config
	authMiddleware   core_http_middleware.Middleware
	rateLimiter      *ratelimiter.RateLimiter
}

type LocationsService interface {
	CreateLocation(
		ctx context.Context,
		location domain.Location,
	) (domain.Location, error)

	GetLocation(
		ctx context.Context,
		id int64,
		userID int64,
	) (domain.Location, error)
}

func NewOpenWeatherHTTPHandler(
	locationsService LocationsService,
	config openweather_service.Config,
	authMiddleware core_http_middleware.Middleware,
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
			Method:      http.MethodGet,
			Path:        "/open-weather/search-location",
			Handler:     h.SearchLocation,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
		{
			Method:      http.MethodPost,
			Path:        "/open-weather/get-weather",
			Handler:     h.GetOpenWeather,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
	}
}
