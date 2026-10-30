package ratelimiter

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

func newRL(t *testing.T, rpm, queue int) *RateLimiter {
	t.Helper()
	logger, _ := zap.NewDevelopment()
	rl, err := NewRateLimiter(
		Config{RequestsPerMinute: rpm, QueueSize: queue},
		&core_logger.Logger{Logger: logger},
	)
	if err != nil {
		t.Fatal(err)
	}
	return rl
}

func drainTokens(t *testing.T, rl *RateLimiter, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := rl.Allow(context.Background()); err != nil {
			t.Fatalf("drain token %d: %v", i, err)
		}
	}
}

// The original bug: tokens exhausted, Close, then Allow -> panic on send to closed channel.
func TestAllowAfterCloseNoPanic(t *testing.T) {
	rl := newRL(t, 1, 10)
	drainTokens(t, rl, 1)
	rl.Close()

	if err := rl.Allow(context.Background()); !errors.Is(err, ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
	if !rl.IsClosed() {
		t.Fatal("IsClosed must be true")
	}
	rl.Close() // idempotent
}

// Requests already waiting when Close is called must be released with ErrClosed,
// not left hanging until their own context is cancelled.
func TestCloseReleasesWaiters(t *testing.T) {
	rl := newRL(t, 1, 50)
	drainTokens(t, rl, 1)

	const n = 20
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() { errs <- rl.Allow(context.Background()) }() // no deadline: would hang forever on the old code
	}
	// let them enqueue
	deadline := time.Now().Add(2 * time.Second)
	for len(rl.queue) < n && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	rl.Close()

	for i := 0; i < n; i++ {
		select {
		case err := <-errs:
			if !errors.Is(err, ErrClosed) {
				t.Fatalf("waiter %d: want ErrClosed, got %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("waiter %d still blocked after Close", i)
		}
	}
}

// Hammer Allow against Close under -race; nothing may panic or hang, and no goroutines may leak.
func TestConcurrentAllowAndClose(t *testing.T) {
	before := runtime.NumGoroutine()

	for iter := 0; iter < 200; iter++ {
		rl := newRL(t, 1, 8)
		drainTokens(t, rl, 1)

		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := rl.Allow(context.Background())
				if err != nil && !errors.Is(err, ErrClosed) && !errors.Is(err, ErrQueueFull) {
					t.Errorf("unexpected error: %v", err)
				}
			}()
		}
		if iter%2 == 0 {
			runtime.Gosched()
		}
		rl.Close()

		done := make(chan struct{})
		go func() { wg.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatalf("iteration %d: Allow calls still blocked after Close", iter)
		}
	}

	time.Sleep(100 * time.Millisecond)
	if after := runtime.NumGoroutine(); after > before+2 {
		t.Fatalf("goroutine leak: before=%d after=%d", before, after)
	}
}

// Sanity: normal queued flow still works (token refill is slow, so use a short-lived context).
func TestQueuedRequestTimesOutNormally(t *testing.T) {
	rl := newRL(t, 1, 5)
	defer rl.Close()
	drainTokens(t, rl, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if err := rl.Allow(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
}
