package sessions_repository_postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
)

func (r *SessionsRepository) DeleteSession(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	DELETE FROM weather.sessions
	WHERE id=$1;
	`

	cmdTag, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("session with id='%s': %w", sessionID, core_errors.ErrNotFound)
	}

	return nil
}
