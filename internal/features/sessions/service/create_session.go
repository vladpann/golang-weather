package sessions_service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (s *SessionsService) CreateSession(
	ctx context.Context,
	userID int64,
) (domain.Session, error) {
	// TODO: validator
	// TODO: заменить 24 * time.Hour на .env

	session := domain.Session{
		ID:        uuid.New(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	session, err := s.sessionsRepository.CreateSession(ctx, session)
	if err != nil {
		return domain.Session{}, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}
