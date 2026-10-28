package sessions_service

import (
	"context"

	"github.com/google/uuid"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

type SessionsService struct {
	config             Config
	sessionsRepository SessionsRepository
}

type SessionsRepository interface {
	CreateSession(
		ctx context.Context,
		session domain.Session,
	) (domain.Session, error)

	GetSession(
		ctx context.Context,
		sessionID uuid.UUID,
	) (domain.Session, error)

	DeleteSession(
		ctx context.Context,
		sessionID uuid.UUID,
	) error

	DeleteExpiredSession(
		ctx context.Context,
	) error
}

func NewSessionsService(
	config Config,
	sessionsRepository SessionsRepository,
) *SessionsService {
	return &SessionsService{
		config:             config,
		sessionsRepository: sessionsRepository,
	}
}
