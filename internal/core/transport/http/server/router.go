package core_http_server

import (
	"net/http"

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
		handler := http.Handler(route.Handler)
		
		// Применяем middleware в обратном порядке
		for i := len(route.Middlewares) - 1; i >= 0; i-- {
			handler = route.Middlewares[i](handler)
		}
		
		r.Method(route.Method, route.Path, handler)
	}
}
