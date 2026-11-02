package openweather_client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	openweather_client "github.com/vladpann/backend/golang-weather/internal/features/openweather/client"
)

func newTestClient(t *testing.T, serverURL string) *openweather_client.Client {
	t.Helper()

	config := openweather_client.Config{
		APIBaseURL:        serverURL,
		APIKey:            "test-api-key",
		Timeout:           5 * time.Second,
		MaxResponseSize:   1048576,
		MaxRetries:        2,
		RetryDelay:        50 * time.Millisecond,
		CacheTTL:          5 * time.Minute,
		CacheCleanupInt:   10 * time.Minute,
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	logger := &core_logger.Logger{Logger: zapLogger}

	client, err := openweather_client.NewClient(config, logger, nil)
	require.NoError(t, err)

	t.Cleanup(func() { client.Close() })

	return client
}

func TestClient_GetWeather_Success(t *testing.T) {
	expected := openweather_client.WeatherResponse{
		Name: "Moscow",
		Coord: openweather_client.Coord{
			Lat: 55.7522,
			Lon: 37.6156,
		},
		Main: openweather_client.Main{
			Temp:     273.15,
			Humidity: 80,
		},
		Sys: openweather_client.Sys{
			Country: "RU",
		},
		Cod: 200,
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		assert.Equal(t, "/data/2.5/weather", r.URL.Path)
		assert.Equal(t, "test-api-key", r.URL.Query().Get("appid"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	require.NoError(t, err)
	assert.Equal(t, "Moscow", result.Name)
	assert.Equal(t, 55.7522, result.Coord.Lat)
	assert.Equal(t, 37.6156, result.Coord.Lon)
	assert.Equal(t, 273.15, result.Main.Temp)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))
}

func TestClient_GetWeather_Cache(t *testing.T) {
	expected := openweather_client.WeatherResponse{
		Name: "Moscow",
		Cod:  200,
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	// First call - should hit the server
	result1, err := client.GetWeather(context.Background(), 55.7522, 37.6156)
	require.NoError(t, err)
	assert.Equal(t, "Moscow", result1.Name)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))

	// Second call - should use cache
	result2, err := client.GetWeather(context.Background(), 55.7522, 37.6156)
	require.NoError(t, err)
	assert.Equal(t, "Moscow", result2.Name)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls), "should not call server again")
}

func TestClient_GetWeather_Deduplication(t *testing.T) {
	expected := openweather_client.WeatherResponse{
		Name: "Moscow",
		Cod:  200,
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	// Make 5 concurrent requests for the same data
	const numRequests = 5
	results := make([]*openweather_client.WeatherResponse, numRequests)
	errors := make([]error, numRequests)

	done := make(chan bool)
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			results[idx], errors[idx] = client.GetWeather(context.Background(), 55.7522, 37.6156)
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// All requests should succeed
	for i := 0; i < numRequests; i++ {
		require.NoError(t, errors[i])
		assert.Equal(t, "Moscow", results[i].Name)
	}

	// But only one call should be made to the server
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls), "deduplication should prevent multiple calls")
}

func TestClient_GetWeather_InvalidAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid API key",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusUnauthorized, upstreamErr.StatusCode)
	assert.ErrorIs(t, upstreamErr.Err, openweather_client.ErrInvalidAPIKey)
	assert.Equal(t, http.StatusBadGateway, upstreamErr.HTTPStatusCode())
	assert.True(t, upstreamErr.IsClientAuth)
}

func TestClient_GetWeather_RateLimitExceeded(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "API rate limit exceeded",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusTooManyRequests, upstreamErr.StatusCode)
	assert.ErrorIs(t, upstreamErr.Err, openweather_client.ErrRateLimitExceeded)
	assert.Equal(t, 1, upstreamErr.RetryAfter)
	assert.Equal(t, http.StatusServiceUnavailable, upstreamErr.HTTPStatusCode())

	// Should retry (MaxRetries = 2, so total 3 attempts)
	assert.EqualValues(t, 3, atomic.LoadInt32(&calls))
}

func TestClient_GetWeather_ServiceUnavailable(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusServiceUnavailable, upstreamErr.StatusCode)
	assert.ErrorIs(t, upstreamErr.Err, openweather_client.ErrServiceUnavailable)
	assert.True(t, upstreamErr.IsRetryable)

	// Should retry
	assert.EqualValues(t, 3, atomic.LoadInt32(&calls))
}

func TestClient_GetWeather_BadGateway_InvalidJSON(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusBadGateway, upstreamErr.StatusCode)
	assert.ErrorIs(t, upstreamErr.Err, openweather_client.ErrBadGateway)
	assert.False(t, upstreamErr.IsRetryable, "invalid JSON should not retry")

	// Should not retry
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))
}

func TestClient_GetWeather_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than client timeout
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result, err := client.GetWeather(ctx, 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	// Context timeout should be returned directly or wrapped
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, openweather_client.ErrGatewayTimeout))
}

func TestClient_GetWeather_RetrySuccess(t *testing.T) {
	expected := openweather_client.WeatherResponse{
		Name: "Moscow",
		Cod:  200,
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&calls, 1)

		// Fail first two attempts, succeed on third
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	require.NoError(t, err)
	assert.Equal(t, "Moscow", result.Name)
	assert.EqualValues(t, 3, atomic.LoadInt32(&calls), "should retry twice and succeed")
}

func TestClient_GetWeather_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "city not found",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.Equal(t, http.StatusNotFound, upstreamErr.StatusCode)
	assert.ErrorIs(t, upstreamErr.Err, openweather_client.ErrNotFound)
	assert.Equal(t, http.StatusNotFound, upstreamErr.HTTPStatusCode())
}

func TestClient_SearchLocation_Success(t *testing.T) {
	expected := []openweather_client.GeocodingResponse{
		{
			Name:    "Moscow",
			Lat:     55.7522,
			Lon:     37.6156,
			Country: "RU",
			State:   "Moscow",
		},
		{
			Name:    "Moscow",
			Lat:     55.8333,
			Lon:     37.6156,
			Country: "RU",
			State:   "Moscow Oblast",
		},
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		assert.Equal(t, "/geo/1.0/direct", r.URL.Path)
		assert.Equal(t, "Moscow", r.URL.Query().Get("q"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "test-api-key", r.URL.Query().Get("appid"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.SearchLocation(context.Background(), "Moscow", 5)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "Moscow", result[0].Name)
	assert.Equal(t, 55.7522, result[0].Lat)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))
}

func TestClient_SearchLocation_Cache(t *testing.T) {
	expected := []openweather_client.GeocodingResponse{
		{Name: "Moscow", Lat: 55.7522, Lon: 37.6156},
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	// First call
	result1, err := client.SearchLocation(context.Background(), "Moscow", 5)
	require.NoError(t, err)
	assert.Len(t, result1, 1)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))

	// Second call - should use cache
	result2, err := client.SearchLocation(context.Background(), "Moscow", 5)
	require.NoError(t, err)
	assert.Len(t, result2, 1)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls), "should not call server again")
}

func TestClient_SearchLocation_EmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]openweather_client.GeocodingResponse{})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.SearchLocation(context.Background(), "NonExistent", 5)

	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestClient_BodySizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Generate a response larger than MaxResponseSize (1MB)
		// Start with valid JSON structure but make it huge
		w.Write([]byte(`{"name":"test","description":"`))
		largeData := make([]byte, 2*1024*1024)
		for i := range largeData {
			largeData[i] = 'a'
		}
		w.Write(largeData)
		w.Write([]byte(`"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	result, err := client.GetWeather(context.Background(), 55.7522, 37.6156)

	assert.Nil(t, result)
	require.Error(t, err)

	// When body exceeds limit, JSON decoder will fail or we'll detect extra bytes
	var upstreamErr *openweather_client.UpstreamError
	require.ErrorAs(t, err, &upstreamErr)
	assert.True(t,
		errors.Is(upstreamErr.Err, openweather_client.ErrBodyTooLarge) ||
			errors.Is(upstreamErr.Err, openweather_client.ErrBadGateway),
		"Expected BodyTooLarge or BadGateway error, got: %v", upstreamErr.Err)
}

func TestClient_ClearCache(t *testing.T) {
	expected := openweather_client.WeatherResponse{
		Name: "Moscow",
		Cod:  200,
	}

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)

	// First call
	_, err := client.GetWeather(context.Background(), 55.7522, 37.6156)
	require.NoError(t, err)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))

	// Clear cache
	client.ClearCache()

	// Second call - should hit server again
	_, err = client.GetWeather(context.Background(), 55.7522, 37.6156)
	require.NoError(t, err)
	assert.EqualValues(t, 2, atomic.LoadInt32(&calls), "should call server after cache clear")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  openweather_client.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: openweather_client.Config{
				APIBaseURL:        "https://api.openweathermap.org",
				APIKey:            "test-key",
				Timeout:           10 * time.Second,
				MaxResponseSize:   1048576,
				MaxRetries:        2,
				RequestsPerMinute: 60,
				QueueSize:         100,
			},
			wantErr: false,
		},
		{
			name: "missing API base URL",
			config: openweather_client.Config{
				APIKey:            "test-key",
				Timeout:           10 * time.Second,
				MaxResponseSize:   1048576,
				RequestsPerMinute: 60,
				QueueSize:         100,
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			config: openweather_client.Config{
				APIBaseURL:        "https://api.openweathermap.org",
				Timeout:           10 * time.Second,
				MaxResponseSize:   1048576,
				RequestsPerMinute: 60,
				QueueSize:         100,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: openweather_client.Config{
				APIBaseURL:        "https://api.openweathermap.org",
				APIKey:            "test-key",
				Timeout:           0,
				MaxResponseSize:   1048576,
				RequestsPerMinute: 60,
				QueueSize:         100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpstreamError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *openweather_client.UpstreamError
		expected string
	}{
		{
			name: "with message",
			err: &openweather_client.UpstreamError{
				StatusCode: 429,
				Message:    "rate limit exceeded",
				Err:        openweather_client.ErrRateLimitExceeded,
			},
			expected: "upstream error (status 429): rate limit exceeded",
		},
		{
			name: "without message",
			err: &openweather_client.UpstreamError{
				StatusCode: 500,
				Err:        fmt.Errorf("internal error"),
			},
			expected: "upstream error (status 500): internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}
