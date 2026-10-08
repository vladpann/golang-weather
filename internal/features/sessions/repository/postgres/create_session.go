package sessions_repository_postgres

import (
	"context"
	"fmt"

	"github.com/vladpann/golang-weather/internal/core/domain"
)

func (r *SessionsRepository) CreateSession(
	ctx context.Context,
	session domain.Session,
) (domain.Session, error) {
	ctx, cancel := context.WithTimeout(ctx, r.pool.OpTimeout())
	defer cancel()

	query := `
		INSERT INTO weather.sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, expires_at
	`

	row := r.pool.QueryRow(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.ExpiresAt,
	)

	var sessionModel SessionModel
	if err := row.Scan(
		&sessionModel.ID,
		&sessionModel.UserID,
		&sessionModel.ExpiresAt,
	); err != nil {
		return domain.Session{}, fmt.Errorf("scan session: %w", err)
	}

	sessionDomain := sessionDomainFromModel(sessionModel)

	return sessionDomain, nil
}
