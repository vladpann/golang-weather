package users_transport_http

import (
	"context"
	"net/http"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_http_server "github.com/vladpann/golang-weather/internal/core/transport/http/server"
)

type UsersHTTPHandler struct {
	usersService UsersService
}

type UsersService interface {
	CreateUser(
		ctx context.Context,
		login string,
		password string,
		repeatPassword string,
	) (domain.User, error)
}

func NewUsersHTTPHandler(
	usersService UsersService,
) *UsersHTTPHandler {
	return &UsersHTTPHandler{
		usersService: usersService,
	}
}

func (h *UsersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodPost,
			Path:    "/auth/sign-up",
			Handler: h.CreateUser,
		},
	}
}
