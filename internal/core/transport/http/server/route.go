package core_http_server

import (
	"net/http"

	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
)

type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	Middlewares []core_http_middleware.Middleware
}
