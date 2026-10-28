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

type SessionsCleaner struct {
	config          Config
	sessionsService SessionsService
	log             *core_logger.Logger
}

func NewSessionsCleaner(
	config Config,
	sessionsService SessionsService,
	log *core_logger.Logger,
) *SessionsCleaner {
	return &SessionsCleaner{
		config:          config,
		sessionsService: sessionsService,
		log:             log,
	}
}

func (c *SessionsCleaner) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	c.log.Info("session cleaner started", zap.Duration("interval", c.config.CleanupInterval))

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
