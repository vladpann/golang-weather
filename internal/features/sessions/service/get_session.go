package sessions_service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
)

func (s *SessionsService) GetSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (domain.Session, error) {
	session, err := s.sessionsRepository.GetSession(ctx, sessionID)
	if err != nil {
		return domain.Session{}, fmt.Errorf("get session from repository: %w", err)
	}

	return session, nil
}
