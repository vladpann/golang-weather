package locations_transport_http

import (
	"context"
	"net/http"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
)

type LocationsHTTPHandler struct {
	locationsService LocationsService
	authMiddleware   core_http_middleware.Middleware
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

	GetLocations(
		ctx context.Context,
		userID int64,
		limit *int,
		offset *int,
	) ([]domain.Location, error)

	DeleteLocation(
		ctx context.Context,
		id int64,
	) error
}

func NewLocationsHTTPHandler(
	locationsService LocationsService,
	authMiddleware core_http_middleware.Middleware,
) *LocationsHTTPHandler {
	return &LocationsHTTPHandler{
		locationsService: locationsService,
		authMiddleware:   authMiddleware,
	}
}

func (h *LocationsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:      http.MethodPost,
			Path:        "/locations",
			Handler:     h.CreateLocation,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
		{
			Method:      http.MethodGet,
			Path:        "/locations",
			Handler:     h.GetLocations,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/locations/{id}",
			Handler:     h.DeleteLocation,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
	}
}
