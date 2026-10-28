package core_http_middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

type contextKey string

const (
	UserIDContextKey    contextKey = "userID"
	SessionIDContextKey contextKey = "sessionID"
)

type SessionsService interface {
	GetSession(
		ctx context.Context,
		sessionID uuid.UUID,
	) (domain.Session, error)
}

func Auth(sessionsService SessionsService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := core_logger.FromContext(ctx)

			cookie, err := r.Cookie("session_id")
			if err != nil {
				log.Debug("no session cookie found")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			sessionID, err := uuid.Parse(cookie.Value)
			if err != nil {
				log.Debug("invalid session ID format")
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}

			session, err := sessionsService.GetSession(ctx, sessionID)
			if err != nil {
				log.Debug("session not found or expired", zap.Error(err))
				http.Error(w, "Session not found or expired", http.StatusUnauthorized)
				return
			}

			ctx = context.WithValue(ctx, UserIDContextKey, session.UserID)
			ctx = context.WithValue(ctx, SessionIDContextKey, sessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(int64)
	return userID, ok
}

func GetSessionIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	sessionID, ok := ctx.Value(SessionIDContextKey).(uuid.UUID)
	return sessionID, ok
}
