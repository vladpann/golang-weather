package sessions_repository_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_postgres_pool "github.com/vladpann/backend/golang-weather/internal/core/repository/postgres/pool"
)

func (r *SessionsRepository) GetSession(
	ctx context.Context,
	sessionID uuid.UUID,
) (domain.Session, error) {
	query := `
		SELECT id, user_id, expires_at
		FROM weather.sessions
		WHERE id = $1 AND expires_at > NOW()
	`

	var model SessionModel
	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&model.ID,
		&model.UserID,
		&model.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, core_postgres_pool.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("session %s: %w", sessionID, core_errors.ErrNotFound)
		}

		return domain.Session{}, fmt.Errorf("get session: %w", err)
	}

	return sessionDomainFromModel(model), nil
}
