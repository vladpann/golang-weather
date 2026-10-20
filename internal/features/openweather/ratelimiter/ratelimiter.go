package ratelimiter

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// queuedRequest represents a request waiting in the queue
type queuedRequest struct {
	ctx       context.Context
	allowChan chan struct{}
	errChan   chan error
}

// RateLimiter manages API request rate limiting using token bucket algorithm
type RateLimiter struct {
	mu             sync.Mutex
	requestsPerMin int
	windowDuration time.Duration
	tokens         int
	lastRefill     time.Time
	queue          chan *queuedRequest
	queueSize      int
	logger         *zap.Logger
	stopChan       chan struct{}
	wg             sync.WaitGroup
	closed         bool // Track if already closed
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(config Config, logger *zap.Logger) (*RateLimiter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	rl := &RateLimiter{
		requestsPerMin: config.RequestsPerMinute,
		windowDuration: time.Minute,
		tokens:         config.RequestsPerMinute,
		lastRefill:     time.Now(),
		queue:          make(chan *queuedRequest, config.QueueSize),
		queueSize:      config.QueueSize,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}

	// Start background queue processor
	rl.wg.Add(1)
	go rl.processQueue()

	logger.Info("rate limiter initialized",
		zap.Int("requests_per_minute", config.RequestsPerMinute),
		zap.Int("queue_size", config.QueueSize))

	return rl, nil
}

// Allow blocks until the request can proceed or context is cancelled
func (rl *RateLimiter) Allow(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Try to acquire token immediately
	if rl.tryAcquireToken() {
		rl.logger.Debug("token acquired immediately")
		return nil
	}

	// Token not available, queue the request
	qr := &queuedRequest{
		ctx:       ctx,
		allowChan: make(chan struct{}),
		errChan:   make(chan error, 1),
	}

	// Try to add to queue
	select {
	case rl.queue <- qr:
		rl.logger.Debug("request queued", zap.Int("queue_size", len(rl.queue)))
	case <-ctx.Done():
		return ctx.Err()
	default:
		rl.logger.Warn("rate limit queue is full")
		return ErrQueueFull
	}

	// Wait for either allow signal or context cancellation
	select {
	case <-qr.allowChan:
		rl.logger.Debug("queued request allowed")
		return nil
	case err := <-qr.errChan:
		return err
	case <-ctx.Done():
		rl.logger.Warn("request timed out while waiting in queue")
		return ctx.Err()
	}
}

// tryAcquireToken attempts to acquire a token immediately
func (rl *RateLimiter) tryAcquireToken() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refillTokens()

	if rl.tokens > 0 {
		rl.tokens--
		rl.logger.Debug("token acquired", zap.Int("remaining_tokens", rl.tokens))
		return true
	}

	return false
}

// refillTokens updates available tokens based on elapsed time
func (rl *RateLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// If a full window has passed, reset to full capacity
	if elapsed >= rl.windowDuration {
		oldTokens := rl.tokens
		rl.tokens = rl.requestsPerMin
		rl.lastRefill = now
		rl.logger.Debug("token bucket refilled",
			zap.Int("old_tokens", oldTokens),
			zap.Int("new_tokens", rl.tokens),
			zap.Duration("elapsed", elapsed))
		return
	}

	// Otherwise, add tokens proportionally to elapsed time
	tokensToAdd := int(float64(rl.requestsPerMin) * elapsed.Seconds() / rl.windowDuration.Seconds())
	if tokensToAdd > 0 {
		oldTokens := rl.tokens
		rl.tokens = min(rl.tokens+tokensToAdd, rl.requestsPerMin)
		rl.lastRefill = now
		rl.logger.Debug("tokens added",
			zap.Int("old_tokens", oldTokens),
			zap.Int("new_tokens", rl.tokens),
			zap.Int("added", tokensToAdd))
	}
}

// processQueue handles queued requests as tokens become available
func (rl *RateLimiter) processQueue() {
	defer rl.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopChan:
			return
		case <-ticker.C:
			rl.processQueuedRequests()
		}
	}
}

// processQueuedRequests attempts to process pending requests
func (rl *RateLimiter) processQueuedRequests() {
	for {
		// Check if there are queued requests
		if len(rl.queue) == 0 {
			return
		}

		// Try to acquire a token
		if !rl.tryAcquireToken() {
			return
		}

		// Get next request from queue
		select {
		case qr := <-rl.queue:
			// Check if context is still valid
			select {
			case <-qr.ctx.Done():
				rl.logger.Warn("queued request context expired, skipping")
				// Return token since we didn't use it
				rl.returnToken()
				continue
			default:
				// Context is valid, allow the request
				close(qr.allowChan)
			}
		default:
			// No request available, return token
			rl.returnToken()
			return
		}
	}
}

// returnToken returns a token to the bucket (called when a queued request is cancelled)
func (rl *RateLimiter) returnToken() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens < rl.requestsPerMin {
		rl.tokens++
		rl.logger.Debug("token returned", zap.Int("remaining_tokens", rl.tokens))
	}
}

// IsClosed returns whether the rate limiter has been closed
func (rl *RateLimiter) IsClosed() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.closed
}

// Close stops the rate limiter and cleans up resources
func (rl *RateLimiter) Close() {
	rl.mu.Lock()
	if rl.closed {
		rl.mu.Unlock()
		return // Already closed
	}
	rl.closed = true
	rl.mu.Unlock()

	close(rl.stopChan)
	rl.wg.Wait()
	close(rl.queue)
}
