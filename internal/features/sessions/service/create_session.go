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
	session := domain.NewSession(
		uuid.New(),
		userID,
		time.Now().Add(s.config.TTL),
	)

	session, err := s.sessionsRepository.CreateSession(ctx, session)
	if err != nil {
		return domain.Session{}, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}
