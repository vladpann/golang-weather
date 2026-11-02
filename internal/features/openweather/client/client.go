package openweather_client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_metrics "github.com/vladpann/backend/golang-weather/internal/core/metrics"
	"go.uber.org/zap"
)

// Client provides access to the OpenWeather API with caching, deduplication, and retry logic
type Client struct {
	config       Config
	httpClient   *http.Client
	cache        *Cache
	deduplicator *Deduplicator
	logger       *core_logger.Logger
	metrics      *core_metrics.Metrics
}

// NewClient creates a new OpenWeather API client
func NewClient(config Config, logger *core_logger.Logger, metrics *core_metrics.Metrics) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	cache := NewCache(config.CacheTTL, config.CacheCleanupInt)
	deduplicator := NewDeduplicator()

	return &Client{
		config:       config,
		httpClient:   httpClient,
		cache:        cache,
		deduplicator: deduplicator,
		logger:       logger,
		metrics:      metrics,
	}, nil
}

// GetWeather retrieves weather data for the specified coordinates
func (c *Client) GetWeather(ctx context.Context, lat, lon float64) (*WeatherResponse, error) {
	cacheKey := fmt.Sprintf("weather:%.4f:%.4f", lat, lon)

	// Check cache first
	if cached, found := c.cache.Get(cacheKey); found {
		c.logger.Debug("cache hit for weather request",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon))
		return cached.(*WeatherResponse), nil
	}

	// Use deduplicator to prevent concurrent requests for the same data
	result, err := c.deduplicator.Do(ctx, cacheKey, func(ctx context.Context) (interface{}, error) {
		query := url.Values{}
		query.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
		query.Set("lon", strconv.FormatFloat(lon, 'f', -1, 64))
		query.Set("appid", c.config.APIKey)

		reqURL := fmt.Sprintf("%s/data/2.5/weather?%s", c.config.APIBaseURL, query.Encode())

		var response WeatherResponse
		start := time.Now()
		err := c.doRequestWithRetry(ctx, reqURL, &response)
		duration := time.Since(start).Seconds()

		// Record upstream metrics
		if c.metrics != nil {
			endpointGroup := "weather"
			if err != nil {
				c.metrics.UpstreamRequestsTotal.WithLabelValues(endpointGroup, "error").Inc()
				c.recordUpstreamError(endpointGroup, err)
			} else {
				c.metrics.UpstreamRequestsTotal.WithLabelValues(endpointGroup, "success").Inc()
			}
			c.metrics.UpstreamRequestDuration.WithLabelValues(endpointGroup).Observe(duration)
		}

		if err != nil {
			return nil, err
		}

		// Cache successful response
		c.cache.Set(cacheKey, &response)

		return &response, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*WeatherResponse), nil
}

// SearchLocation searches for locations by city name
func (c *Client) SearchLocation(ctx context.Context, cityName string, limit int) ([]GeocodingResponse, error) {
	if limit <= 0 {
		limit = 5
	}

	cacheKey := fmt.Sprintf("geocoding:%s:%d", cityName, limit)

	// Check cache first
	if cached, found := c.cache.Get(cacheKey); found {
		c.logger.Debug("cache hit for geocoding request",
			zap.String("city", cityName))
		return cached.([]GeocodingResponse), nil
	}

	// Use deduplicator to prevent concurrent requests for the same data
	result, err := c.deduplicator.Do(ctx, cacheKey, func(ctx context.Context) (interface{}, error) {
		query := url.Values{}
		query.Set("q", cityName)
		query.Set("limit", strconv.Itoa(limit))
		query.Set("appid", c.config.APIKey)

		reqURL := fmt.Sprintf("%s/geo/1.0/direct?%s", c.config.APIBaseURL, query.Encode())

		var response []GeocodingResponse
		start := time.Now()
		err := c.doRequestWithRetry(ctx, reqURL, &response)
		duration := time.Since(start).Seconds()

		// Record upstream metrics
		if c.metrics != nil {
			endpointGroup := "geocoding"
			if err != nil {
				c.metrics.UpstreamRequestsTotal.WithLabelValues(endpointGroup, "error").Inc()
				c.recordUpstreamError(endpointGroup, err)
			} else {
				c.metrics.UpstreamRequestsTotal.WithLabelValues(endpointGroup, "success").Inc()
			}
			c.metrics.UpstreamRequestDuration.WithLabelValues(endpointGroup).Observe(duration)
		}

		if err != nil {
			return nil, err
		}

		// Cache successful response
		c.cache.Set(cacheKey, response)

		return response, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]GeocodingResponse), nil
}

// doRequestWithRetry executes an HTTP request with retry logic
func (c *Client) doRequestWithRetry(ctx context.Context, url string, target interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryDelay * time.Duration(attempt)):
			}

			c.logger.Debug("retrying request",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", c.config.MaxRetries))
		}

		err := c.doRequest(ctx, url, target)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		var upstreamErr *UpstreamError
		if errors.As(err, &upstreamErr) {
			// Don't retry on client errors or auth issues
			if !upstreamErr.IsRetryable {
				return err
			}

			// If upstream provides Retry-After, use it
			if upstreamErr.RetryAfter > 0 && attempt < c.config.MaxRetries {
				c.logger.Debug("upstream requested retry after delay",
					zap.Int("retry_after_seconds", upstreamErr.RetryAfter))

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(upstreamErr.RetryAfter) * time.Second):
				}
			}
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			// Don't retry on context errors
			return err
		}

		// For network errors, check if we should retry
		var netErr net.Error
		if errors.As(err, &netErr) && !netErr.Timeout() {
			// Connection refused, etc. - don't retry
			return err
		}
	}

	return lastErr
}

// doRequest executes a single HTTP request
func (c *Client) doRequest(ctx context.Context, url string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check for timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return &UpstreamError{
				StatusCode:  http.StatusGatewayTimeout,
				Err:         ErrGatewayTimeout,
				IsRetryable: true,
			}
		}

		// Network error
		return &UpstreamError{
			StatusCode:  http.StatusBadGateway,
			Message:     "network error",
			Err:         err,
			IsRetryable: true,
		}
	}
	defer resp.Body.Close()

	// Check for non-OK status
	if resp.StatusCode != http.StatusOK {
		return c.handleErrorResponse(resp)
	}

	// Limit response body size
	limitedReader := io.LimitReader(resp.Body, c.config.MaxResponseSize)

	// Decode response
	if err := json.NewDecoder(limitedReader).Decode(target); err != nil {
		c.logger.Error("failed to decode response",
			zap.Error(err),
			zap.Int("status", resp.StatusCode))

		return &UpstreamError{
			StatusCode:  http.StatusBadGateway,
			Message:     "invalid JSON response",
			Err:         ErrBadGateway,
			IsRetryable: false,
		}
	}

	// Check if we hit the size limit
	buf := make([]byte, 1)
	if n, _ := limitedReader.Read(buf); n > 0 {
		c.logger.Warn("response body exceeded size limit",
			zap.Int64("max_size", c.config.MaxResponseSize))

		return &UpstreamError{
			StatusCode:  http.StatusBadGateway,
			Err:         ErrBodyTooLarge,
			IsRetryable: false,
		}
	}

	return nil
}

// handleErrorResponse processes error responses from the upstream API
func (c *Client) handleErrorResponse(resp *http.Response) error {
	// Try to read error message from response body
	limitedReader := io.LimitReader(resp.Body, 4096) // Limit error body to 4KB
	bodyBytes, _ := io.ReadAll(limitedReader)
	bodyStr := string(bodyBytes)

	// Parse Retry-After header if present
	retryAfter := 0
	if retryAfterHeader := resp.Header.Get("Retry-After"); retryAfterHeader != "" {
		if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
			retryAfter = seconds
		}
	}

	c.logger.Debug("upstream error response",
		zap.Int("status", resp.StatusCode),
		zap.String("body", bodyStr),
		zap.Int("retry_after", retryAfter))

	upstreamErr := &UpstreamError{
		StatusCode: resp.StatusCode,
		Message:    strings.TrimSpace(bodyStr),
		RetryAfter: retryAfter,
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		upstreamErr.Err = ErrBadRequest
		upstreamErr.IsRetryable = false

	case http.StatusUnauthorized, http.StatusForbidden:
		upstreamErr.Err = ErrInvalidAPIKey
		upstreamErr.IsRetryable = false
		upstreamErr.IsClientAuth = true
		c.logger.Error("upstream authentication failed - check API key",
			zap.Int("status", resp.StatusCode))

	case http.StatusNotFound:
		upstreamErr.Err = ErrNotFound
		upstreamErr.IsRetryable = false

	case http.StatusTooManyRequests:
		upstreamErr.Err = ErrRateLimitExceeded
		upstreamErr.IsRetryable = true

	case http.StatusInternalServerError, http.StatusBadGateway:
		upstreamErr.Err = ErrServiceUnavailable
		upstreamErr.IsRetryable = true

	case http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		upstreamErr.Err = ErrServiceUnavailable
		upstreamErr.IsRetryable = true

	default:
		upstreamErr.Err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		upstreamErr.IsRetryable = resp.StatusCode >= 500
	}

	return upstreamErr
}

// ClearCache clears all cached data
func (c *Client) ClearCache() {
	c.cache.Clear()
}

// recordUpstreamError categorizes and records upstream error metrics
func (c *Client) recordUpstreamError(endpointGroup string, err error) {
	if c.metrics == nil {
		return
	}

	var upstreamErr *UpstreamError
	if errors.As(err, &upstreamErr) {
		errorType := "server_error"
		if upstreamErr.IsClientAuth {
			errorType = "auth"
		} else if errors.Is(upstreamErr.Err, ErrRateLimitExceeded) {
			errorType = "rate_limit"
		} else if errors.Is(upstreamErr.Err, ErrGatewayTimeout) {
			errorType = "timeout"
		} else if upstreamErr.StatusCode >= 500 {
			errorType = "server_error"
		} else if upstreamErr.StatusCode >= 400 {
			errorType = "client_error"
		}
		c.metrics.UpstreamErrors.WithLabelValues(endpointGroup, errorType).Inc()
	} else if errors.Is(err, context.DeadlineExceeded) {
		c.metrics.UpstreamErrors.WithLabelValues(endpointGroup, "timeout").Inc()
	} else {
		c.metrics.UpstreamErrors.WithLabelValues(endpointGroup, "network").Inc()
	}
}

// Close releases resources used by the client
func (c *Client) Close() {
	c.cache.Close()
}
