package sessions_repository_postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/vladpann/golang-weather/internal/core/domain"
)

type SessionModel struct {
	ID        uuid.UUID
	UserID    int64
	ExpiresAt time.Time
}

func sessionDomainFromModel(session SessionModel) domain.Session {
	return domain.NewSession(
		session.ID,
		session.UserID,
		session.ExpiresAt,
	)
}
