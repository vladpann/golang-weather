package sessions_service

import (
	"context"
	"fmt"
)

func (s *SessionsService) DeleteExpiredSession(
	ctx context.Context,
) error {
	err := s.sessionsRepository.DeleteExpiredSession(ctx)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}
