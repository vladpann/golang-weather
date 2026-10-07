package core_http_server

import (
	"github.com/go-chi/chi/v5"
)

type ApiVersion string

var (
	ApiVersion1 = ApiVersion("v1")
)

type APIVersionRouter struct {
	chi.Router
	apiVersion ApiVersion
}

func NewAPIVersionRouter(
	apiVersion ApiVersion,
) *APIVersionRouter {
	r := chi.NewRouter()

	return &APIVersionRouter{
		Router:     r,
		apiVersion: apiVersion,
	}
}

func (r *APIVersionRouter) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		r.Method(route.Method, route.Path, route.Handler)
	}
}
