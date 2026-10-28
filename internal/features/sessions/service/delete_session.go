package sessions_service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *SessionsService) DeleteSession(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	err := s.sessionsRepository.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
