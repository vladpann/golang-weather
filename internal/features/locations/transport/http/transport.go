package locations_transport_http

import (
	"context"
	"net/http"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
)

type LocationsHTTPHandler struct {
	locationsService LocationsService
	authMiddleware   func(http.Handler) http.Handler
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
	authMiddleware func(http.Handler) http.Handler,
) *LocationsHTTPHandler {
	return &LocationsHTTPHandler{
		locationsService: locationsService,
		authMiddleware:   authMiddleware,
	}
}

func (h *LocationsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:      http.MethodGet,
			Path:        "/locations",
			Handler:     h.GetLocations,
			Middlewares: []func(http.Handler) http.Handler{h.authMiddleware},
		},
		{
			Method:      http.MethodDelete,
			Path:        "/locations/{id}",
			Handler:     h.DeleteLocation,
			Middlewares: []func(http.Handler) http.Handler{h.authMiddleware},
		},
	}
}
