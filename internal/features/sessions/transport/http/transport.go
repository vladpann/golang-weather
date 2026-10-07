package sessions_transport_http

import (
	"context"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

type SessionsHTTPHandler struct {
	sessionsService SessionsService
}

type SessionsService interface {
	CreateSession(
		ctx context.Context,
		userID int64,
	) (domain.Session, error)
}

func NewSessionHTTPHandler(
	sessionsService SessionsService,
) *SessionsHTTPHandler {
	return &SessionsHTTPHandler{
		sessionsService: sessionsService,
	}
}
