package locations_transport_http

import (
	"net/http"

	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
	locations_service "github.com/vladpann/golang-weather/internal/features/locations/service"
)

type LocationsHTTPHandler struct {
	locationsService *locations_service.LocationsService
	authMiddleware   func(http.Handler) http.Handler
}

func NewLocationsHTTPHandler(
	locationsService *locations_service.LocationsService,
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
