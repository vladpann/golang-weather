package sessions_service

import (
	"context"

	"github.com/google/uuid"
	"github.com/vladpann/golang-weather/internal/core/domain"
)

type SessionsService struct {
	sessionsRepository SessionsRepository
	config             Config
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
}

func NewSessionsService(
	sessionsRepository SessionsRepository,
	config Config,
) *SessionsService {
	return &SessionsService{
		sessionsRepository: sessionsRepository,
		config:             config,
	}
}
