package sessions_repository_postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
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
		return domain.Session{}, fmt.Errorf("query row: %w", err)
	}

	return sessionDomainFromModel(model), nil
}
