package sessions_worker

import (
	"context"
	"time"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

type SessionsService interface {
	DeleteExpiredSession(ctx context.Context) error
}

type SessionCleaner struct {
	config          Config
	sessionsService SessionsService
	log             *core_logger.Logger
}

func NewSessionCleaner(
	config Config,
	sessionsService SessionsService,
	log *core_logger.Logger,
) *SessionCleaner {
	return &SessionCleaner{
		config:          config,
		sessionsService: sessionsService,
		log:             log,
	}
}

func (c *SessionCleaner) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.config.Cleanup_Inverval)
	defer ticker.Stop()

	c.log.Info("session cleaner started", zap.Duration("interval", c.config.Cleanup_Inverval))

	for {
		select {
		case <-ctx.Done():
			c.log.Info("session cleaner stopped")
			return nil

		case <-ticker.C:
			if err := c.sessionsService.DeleteExpiredSession(ctx); err != nil {
				c.log.Error("delete expired sessions", zap.Error(err))
				continue
			}
			c.log.Debug("expired sessions have been cleared")
		}
	}
}
