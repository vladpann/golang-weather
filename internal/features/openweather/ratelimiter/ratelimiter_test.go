package ratelimiter_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
)

func newTestRateLimiter(t *testing.T, requestsPerMin, queueSize int) *ratelimiter.RateLimiter {
	t.Helper()

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	config := ratelimiter.Config{
		RequestsPerMinute: requestsPerMin,
		QueueSize:         queueSize,
	}

	rl, err := ratelimiter.NewRateLimiter(config, logger)
	require.NoError(t, err)

	t.Cleanup(func() {
		rl.Close()
	})

	return rl
}

func TestRateLimiter_AllowImmediate(t *testing.T) {
	rl := newTestRateLimiter(t, 60, 10)

	ctx := context.Background()

	// First request should be allowed immediately
	err := rl.Allow(ctx)
	assert.NoError(t, err)
}

func TestRateLimiter_MultipleRequestsWithinLimit(t *testing.T) {
	requestsPerMin := 10
	rl := newTestRateLimiter(t, requestsPerMin, 10)

	ctx := context.Background()

	// All requests within limit should succeed
	for i := 0; i < requestsPerMin; i++ {
		err := rl.Allow(ctx)
		assert.NoError(t, err, "request %d should succeed", i)
	}
}

func TestRateLimiter_ExceedLimit_Queued(t *testing.T) {
	requestsPerMin := 60 // Higher rate for faster token refill
	queueSize := 3
	rl := newTestRateLimiter(t, requestsPerMin, queueSize)

	// Use up all tokens
	for i := 0; i < requestsPerMin; i++ {
		err := rl.Allow(context.Background())
		require.NoError(t, err)
	}

	// Next requests should be queued
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var successCount int32

	// Try to send more requests than queue size
	for i := 0; i < queueSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := rl.Allow(ctx)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// Some requests should have been processed from the queue
	// (tokens refill over time, so at least some should succeed)
	assert.Greater(t, atomic.LoadInt32(&successCount), int32(0), "some queued requests should succeed")
}

func TestRateLimiter_QueueFull(t *testing.T) {
	requestsPerMin := 5
	queueSize := 2
	rl := newTestRateLimiter(t, requestsPerMin, queueSize)

	ctx := context.Background()

	// Use up all tokens
	for i := 0; i < requestsPerMin; i++ {
		err := rl.Allow(ctx)
		require.NoError(t, err)
	}

	// Fill the queue
	var wg sync.WaitGroup
	longCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < queueSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Allow(longCtx)
		}()
	}

	// Give time for queue to fill
	time.Sleep(100 * time.Millisecond)

	// Next request should fail with ErrQueueFull
	err := rl.Allow(ctx)
	assert.ErrorIs(t, err, ratelimiter.ErrQueueFull)

	cancel() // Clean up queued requests
	wg.Wait()
}

func TestRateLimiter_ContextCanceled(t *testing.T) {
	rl := newTestRateLimiter(t, 5, 10)

	// Use up all tokens
	for i := 0; i < 5; i++ {
		err := rl.Allow(context.Background())
		require.NoError(t, err)
	}

	// Cancel context before request
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := rl.Allow(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestRateLimiter_ContextTimeout(t *testing.T) {
	rl := newTestRateLimiter(t, 5, 10)

	// Use up all tokens
	for i := 0; i < 5; i++ {
		err := rl.Allow(context.Background())
		require.NoError(t, err)
	}

	// Context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := rl.Allow(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	requestsPerMin := 60 // Use standard rate
	rl := newTestRateLimiter(t, requestsPerMin, 10)

	ctx := context.Background()

	// Use up all tokens
	for i := 0; i < requestsPerMin; i++ {
		err := rl.Allow(ctx)
		require.NoError(t, err)
	}

	// Wait for tokens to refill (proportional refill happens over time)
	// At 60 requests/min, we get 1 token per second
	time.Sleep(1200 * time.Millisecond)

	// Should be able to make a request now (tokens refilled proportionally)
	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := rl.Allow(ctx2)
	assert.NoError(t, err, "should succeed after token refill")
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	requestsPerMin := 20
	numGoroutines := 50
	rl := newTestRateLimiter(t, requestsPerMin, 100)

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := rl.Allow(ctx)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// All requests should complete (either succeed or fail gracefully)
	assert.Equal(t, int32(numGoroutines), successCount+errorCount)

	// At least some requests should succeed
	assert.Greater(t, successCount, int32(0))
}

func TestRateLimiter_Close(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	config := ratelimiter.Config{
		RequestsPerMinute: 60,
		QueueSize:         10,
	}

	rl, err := ratelimiter.NewRateLimiter(config, logger)
	require.NoError(t, err)

	// Make a request to ensure it's working
	err = rl.Allow(context.Background())
	assert.NoError(t, err)

	// Close the rate limiter
	rl.Close()

	// Verify it's closed
	assert.True(t, rl.IsClosed())

	// Close again should be safe (idempotent)
	rl.Close()
	assert.True(t, rl.IsClosed())
}

func TestRateLimiter_ConfigValidation(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	tests := []struct {
		name      string
		config    ratelimiter.Config
		wantError bool
	}{
		{
			name: "valid config",
			config: ratelimiter.Config{
				RequestsPerMinute: 60,
				QueueSize:         100,
			},
			wantError: false,
		},
		{
			name: "zero requests per minute",
			config: ratelimiter.Config{
				RequestsPerMinute: 0,
				QueueSize:         100,
			},
			wantError: true,
		},
		{
			name: "negative requests per minute",
			config: ratelimiter.Config{
				RequestsPerMinute: -10,
				QueueSize:         100,
			},
			wantError: true,
		},
		{
			name: "negative queue size",
			config: ratelimiter.Config{
				RequestsPerMinute: 60,
				QueueSize:         -10,
			},
			wantError: true,
		},
		{
			name: "zero queue size is invalid",
			config: ratelimiter.Config{
				RequestsPerMinute: 60,
				QueueSize:         0,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl, err := ratelimiter.NewRateLimiter(tt.config, logger)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, rl)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rl)
				if rl != nil {
					rl.Close()
				}
			}
		})
	}
}

func TestRateLimiter_QueueProcessing(t *testing.T) {
	requestsPerMin := 60 // Higher rate for faster refill
	queueSize := 5
	rl := newTestRateLimiter(t, requestsPerMin, queueSize)

	// Use all tokens
	for i := 0; i < requestsPerMin; i++ {
		err := rl.Allow(context.Background())
		require.NoError(t, err)
	}

	// Queue several requests
	var wg sync.WaitGroup
	var processedCount int32

	// Longer timeout to allow token refill (at least 1 second for proportional refill)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < queueSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := rl.Allow(ctx)
			if err == nil {
				atomic.AddInt32(&processedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// At least some requests should have been processed from the queue
	// as tokens refill over time
	t.Logf("Processed %d requests from queue", processedCount)
	assert.Greater(t, processedCount, int32(0), "queue should process requests as tokens refill")
}
