package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    int64
	ExpiresAt time.Time
}

func NewSession(
	id uuid.UUID,
	userID int64,
	expiresAt time.Time,
) Session {
	return Session{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
}
