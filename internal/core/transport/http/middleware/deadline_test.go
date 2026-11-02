package core_http_middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestDeadline_WithinDeadline(t *testing.T) {
	timeout := 100 * time.Millisecond

	handler := RequestDeadline(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has deadline
		deadline, ok := r.Context().Deadline()
		require.True(t, ok, "context should have deadline")
		assert.True(t, time.Until(deadline) > 0, "deadline should be in the future")
		assert.True(t, time.Until(deadline) <= timeout, "deadline should be <= timeout")

		// Simulate some work (less than timeout)
		time.Sleep(10 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequestDeadline_ExceedsDeadline(t *testing.T) {
	timeout := 50 * time.Millisecond

	handler := RequestDeadline(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Simulate work that exceeds deadline
		select {
		case <-time.After(200 * time.Millisecond):
			t.Fatal("should not reach here - context should cancel first")
		case <-ctx.Done():
			// Context cancelled due to deadline
			assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			w.WriteHeader(http.StatusRequestTimeout)
		}
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestTimeout, rr.Code)
}

func TestRequestDeadline_ZeroTimeout(t *testing.T) {
	timeout := time.Duration(0) // No deadline

	handler := RequestDeadline(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not have deadline when timeout is 0
		_, ok := r.Context().Deadline()
		assert.False(t, ok, "context should not have deadline when timeout is 0")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequestDeadline_NegativeTimeout(t *testing.T) {
	timeout := -1 * time.Second // Negative timeout

	handler := RequestDeadline(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not have deadline when timeout is negative
		_, ok := r.Context().Deadline()
		assert.False(t, ok, "context should not have deadline when timeout is negative")

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequestDeadline_PropagatesExistingContext(t *testing.T) {
	timeout := 100 * time.Millisecond

	type contextKey string
	key := contextKey("test-key")
	value := "test-value"

	handler := RequestDeadline(timeout)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that existing context values are preserved
		val := r.Context().Value(key)
		assert.Equal(t, value, val)

		w.WriteHeader(http.StatusOK)
	}))

	// Create request with existing context
	ctx := context.WithValue(context.Background(), key, value)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
