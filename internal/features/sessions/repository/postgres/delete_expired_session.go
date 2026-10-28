package sessions_repository_postgres

import (
	"context"
	"fmt"
	"time"
)

func (r *SessionsRepository) DeleteExpiredSession(
	ctx context.Context,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
	DELETE FROM weather.sessions
	WHERE expires_at <= $1;
	`

	_, err := r.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}
