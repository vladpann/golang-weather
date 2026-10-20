package users_transport_http

import (
	"context"
	"net/http"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_http_middleware "github.com/vladpann/backend/golang-weather/internal/core/transport/http/middleware"
	core_http_server "github.com/vladpann/backend/golang-weather/internal/core/transport/http/server"
)

type UsersHTTPHandler struct {
	usersService    UsersService
	sessionsService SessionsService
	authMiddleware  core_http_middleware.Middleware
}

type UsersService interface {
	CreateUser(
		ctx context.Context,
		login string,
		password string,
		repeatPassword string,
	) (domain.User, error)

	GetUser(
		ctx context.Context,
		login string,
		password string,
	) (domain.User, error)

	GetUserByID(
		ctx context.Context,
		userID int64,
	) (domain.User, error)
}

type SessionsService interface {
	CreateSession(
		ctx context.Context,
		userID int64,
	) (domain.Session, error)
}

func NewUsersHTTPHandler(
	usersService UsersService,
	sessionsService SessionsService,
	authMiddleware core_http_middleware.Middleware,
) *UsersHTTPHandler {
	return &UsersHTTPHandler{
		usersService:    usersService,
		sessionsService: sessionsService,
		authMiddleware:  authMiddleware,
	}
}

func (h *UsersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		{
			Method:  http.MethodPost,
			Path:    "/auth/sign-up",
			Handler: h.CreateUser,
		},
		{
			Method:  http.MethodPost,
			Path:    "/auth/sign-in",
			Handler: h.SignInUser,
		},
		{
			Method:  http.MethodPost,
			Path:    "/auth/sign-out",
			Handler: h.SignOutUser,
		},
		{
			Method:      http.MethodGet,
			Path:        "/auth/check",
			Handler:     h.CheckAuth,
			Middlewares: []core_http_middleware.Middleware{h.authMiddleware},
		},
	}
}
