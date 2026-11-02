package core_http_middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func newTestLogger(t *testing.T) *core_logger.Logger {
	t.Helper()
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatal(err)
	}
	return &core_logger.Logger{Logger: logger}
}

func TestLoadShed_WithinLimit(t *testing.T) {
	config := LoadShedConfig{
		MaxConcurrentRequests: 10,
	}
	logger := newTestLogger(t)

	handler := LoadShed(config, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond) // Simulate work
		w.WriteHeader(http.StatusOK)
	}))

	// Send 5 concurrent requests (within limit of 10)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}()
	}

	wg.Wait()
}

func TestLoadShed_ExceedsLimit(t *testing.T) {
	config := LoadShedConfig{
		MaxConcurrentRequests: 5,
	}
	logger := newTestLogger(t)

	var blockedCount int32
	blockChan := make(chan struct{})

	handler := LoadShed(config, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&blockedCount, 1)
		<-blockChan // Block until test releases
		w.WriteHeader(http.StatusOK)
	}))

	// Fill up to capacity (5 requests)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
		}()
	}

	// Wait for all 5 to be blocked
	for atomic.LoadInt32(&blockedCount) < 5 {
		time.Sleep(1 * time.Millisecond)
	}

	// 6th request should be rejected
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Equal(t, "5", rr.Header().Get("Retry-After"))

	// Release blocked requests
	close(blockChan)
	wg.Wait()
}

func TestLoadShed_Disabled(t *testing.T) {
	config := LoadShedConfig{
		MaxConcurrentRequests: 0, // Disabled
	}
	logger := newTestLogger(t)

	var requestCount int32

	handler := LoadShed(config, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	// Send many concurrent requests (should all be accepted)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(100), atomic.LoadInt32(&requestCount))
}

func TestLoadShed_CorrectlyDecrementsOnCompletion(t *testing.T) {
	config := LoadShedConfig{
		MaxConcurrentRequests: 2,
	}
	logger := newTestLogger(t)

	handler := LoadShed(config, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	// First 2 requests should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	rr1 := httptest.NewRecorder()
	
	req2 := httptest.NewRequest("GET", "/test", nil)
	rr2 := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		handler.ServeHTTP(rr1, req1)
	}()

	go func() {
		defer wg.Done()
		handler.ServeHTTP(rr2, req2)
	}()

	// Wait a bit for them to start
	time.Sleep(10 * time.Millisecond)

	// 3rd request should be rejected
	req3 := httptest.NewRequest("GET", "/test", nil)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req3)
	assert.Equal(t, http.StatusServiceUnavailable, rr3.Code)

	// Wait for first 2 to complete
	wg.Wait()
	assert.Equal(t, http.StatusOK, rr1.Code)
	assert.Equal(t, http.StatusOK, rr2.Code)

	// Now another request should succeed (slot freed)
	req4 := httptest.NewRequest("GET", "/test", nil)
	rr4 := httptest.NewRecorder()
	handler.ServeHTTP(rr4, req4)
	assert.Equal(t, http.StatusOK, rr4.Code)
}
