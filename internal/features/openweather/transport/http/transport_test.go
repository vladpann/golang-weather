package openweather_transport_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vladpann/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/golang-weather/internal/core/logger"
	core_http_middleware "github.com/vladpann/golang-weather/internal/core/transport/http/middleware"
	"github.com/vladpann/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/golang-weather/internal/features/openweather/service"
	openweather_transport_http "github.com/vladpann/golang-weather/internal/features/openweather/transport/http"
)

// MockLocationsService мокирует сервис локаций
type MockLocationsService struct {
	mock.Mock
}

func (m *MockLocationsService) CreateLocation(ctx context.Context, location domain.Location) (domain.Location, error) {
	args := m.Called(ctx, location)
	return args.Get(0).(domain.Location), args.Error(1)
}

func (m *MockLocationsService) GetLocation(ctx context.Context, id int64, userID int64) (domain.Location, error) {
	args := m.Called(ctx, id, userID)
	return args.Get(0).(domain.Location), args.Error(1)
}

// setupTestHandler создает handler с моками для тестирования
func setupTestHandler(t *testing.T, config openweather_service.Config) (*openweather_transport_http.OpenWeatherHTTPHandler, *MockLocationsService, *ratelimiter.RateLimiter, *core_logger.Logger) {
	t.Helper()

	mockLocationsService := new(MockLocationsService)

	// Создаем настоящий rate limiter
	rateLimiterConfig := ratelimiter.Config{
		RequestsPerMinute: config.RequestsPerMinute,
		QueueSize:         config.QueueSize,
	}
	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	rateLimiter, err := ratelimiter.NewRateLimiter(rateLimiterConfig, zapLogger)
	require.NoError(t, err)

	// Создаем logger обертку
	logger := &core_logger.Logger{Logger: zapLogger}

	// Создаем noop auth middleware для тестов
	noopAuthMiddleware := func(next http.Handler) http.Handler { return next }

	handler := openweather_transport_http.NewOpenWeatherHTTPHandler(
		mockLocationsService,
		config,
		noopAuthMiddleware,
		rateLimiter,
	)

	return handler, mockLocationsService, rateLimiter, logger
}

// TestGetOpenWeatherLocation_Success тестирует успешное получение локации
func TestGetOpenWeatherLocation_Success(t *testing.T) {
	// Подготавливаем JSON ответ от OpenWeather API
	mockAPIResponse := openweather_transport_http.OpenWeatherDTOResponse{
		Coord: openweather_transport_http.Coord{
			Lon: 37.6156,
			Lat: 55.7522,
		},
		Name: "Moscow",
		Cod:  200,
	}

	responseBody, err := json.Marshal(mockAPIResponse)
	require.NoError(t, err)

	// Создаем тестовый HTTP сервер
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем параметры запроса
		assert.Contains(t, r.URL.Query().Get("q"), "Moscow")
		assert.Equal(t, "test-api-key", r.URL.Query().Get("appid"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}))
	defer testServer.Close()

	// Создаем config с тестовым сервером
	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	// Мокируем CreateLocation
	expectedLocation := domain.Location{
		ID:        1,
		UserID:    123,
		Name:      "Moscow",
		Latitude:  55.7522,
		Longitude: 37.6156,
	}
	mockLocationsService.On("CreateLocation", mock.Anything, mock.MatchedBy(func(loc domain.Location) bool {
		return loc.Name == "Moscow" && loc.UserID == int64(123)
	})).Return(expectedLocation, nil)

	// Подготавливаем HTTP request
	requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
		CityName: "Moscow",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем контекст с логгером и userID
	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeatherLocation(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var response domain.Location
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Moscow", response.Name)
	assert.Equal(t, 55.7522, response.Latitude)
	assert.Equal(t, 37.6156, response.Longitude)
	assert.Equal(t, int64(123), response.UserID)

	mockLocationsService.AssertExpectations(t)
}

// TestGetOpenWeatherLocation_APIError_404 тестирует обработку 404 ошибки от API
func TestGetOpenWeatherLocation_APIError_404(t *testing.T) {
	// Создаем тестовый HTTP сервер, который вернет 404
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"cod":"404","message":"city not found"}`))
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
		CityName: "NonExistentCity",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeatherLocation(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")

	// Проверяем, что CreateLocation НЕ был вызван
	mockLocationsService.AssertNotCalled(t, "CreateLocation", mock.Anything, mock.Anything)
}

// TestGetOpenWeatherLocation_APIError_503 тестирует обработку 503 ошибки от API
func TestGetOpenWeatherLocation_APIError_503(t *testing.T) {
	// Создаем тестовый HTTP сервер, который вернет 503
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"message":"Service Unavailable"}`))
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
		CityName: "Moscow",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeatherLocation(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")

	mockLocationsService.AssertNotCalled(t, "CreateLocation", mock.Anything, mock.Anything)
}

// TestGetOpenWeatherLocation_InvalidJSON тестирует обработку невалидного JSON от API
func TestGetOpenWeatherLocation_InvalidJSON(t *testing.T) {
	// Создаем тестовый HTTP сервер, который вернет невалидный JSON
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid json`))
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
		CityName: "Moscow",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeatherLocation(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	mockLocationsService.AssertNotCalled(t, "CreateLocation", mock.Anything, mock.Anything)
}

// TestGetOpenWeatherLocation_MissingUserID тестирует обработку отсутствия userID
func TestGetOpenWeatherLocation_MissingUserID(t *testing.T) {
	mockAPIResponse := openweather_transport_http.OpenWeatherDTOResponse{
		Coord: openweather_transport_http.Coord{
			Lon: 37.6156,
			Lat: 55.7522,
		},
		Name: "Moscow",
		Cod:  200,
	}

	responseBody, err := json.Marshal(mockAPIResponse)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
		CityName: "Moscow",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем logger, но НЕ добавляем userID в контекст
	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	// userID намеренно НЕ добавляется для теста
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeatherLocation(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)

	var response map[string]interface{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Contains(t, response, "error")
	assert.Contains(t, response, "message")
	assert.Equal(t, "user not authenticated", response["message"])

	mockLocationsService.AssertNotCalled(t, "CreateLocation", mock.Anything, mock.Anything)
}

// TestGetOpenWeather_Success тестирует успешное получение погоды по координатам
func TestGetOpenWeather_Success(t *testing.T) {
	mockAPIResponse := openweather_transport_http.OpenWeatherDTOResponse{
		Coord: openweather_transport_http.Coord{
			Lon: 37.6156,
			Lat: 55.7522,
		},
		Weather: []openweather_transport_http.Weather{
			{
				ID:          800,
				Main:        "Clear",
				Description: "clear sky",
				Icon:        "01d",
			},
		},
		Main: openweather_transport_http.Main{
			Temp:      273.15,
			FeelsLike: 270.0,
			Pressure:  1013,
			Humidity:  80,
		},
		Name: "Moscow",
		Cod:  200,
	}

	responseBody, err := json.Marshal(mockAPIResponse)
	require.NoError(t, err)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.URL.Query().Get("lat"))
		assert.NotEmpty(t, r.URL.Query().Get("lon"))
		assert.Equal(t, "test-api-key", r.URL.Query().Get("appid"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, _, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherRequest{
		Latitude:  55.7522,
		Longitude: 37.6156,
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-weather", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeather(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	var response openweather_transport_http.GetOpenWeatherResponse
	err = json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Moscow", response.Name)
	assert.Equal(t, 37.6156, response.Coord.Lon)
	assert.Equal(t, 55.7522, response.Coord.Lat)
	assert.Equal(t, 273.15, response.Main.Temp)
	assert.Equal(t, "Clear", response.Weather[0].Main)
}

// TestGetOpenWeather_APIError_400 тестирует обработку 400 ошибки от API
func TestGetOpenWeather_APIError_400(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"cod":"400","message":"Invalid coordinates"}`))
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	handler, _, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	requestBody := openweather_transport_http.GetOpenWeatherRequest{
		Latitude:  999.0,
		Longitude: 999.0,
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-weather", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ctx := req.Context()
	ctx = core_logger.ToContext(ctx, logger)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	// Act
	handler.GetOpenWeather(rr, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// TestRateLimitBehavior тестирует работу rate limiter
func TestRateLimitBehavior(t *testing.T) {
	t.Skip("Этот тест требует времени для проверки rate limiting - запускать вручную если нужно")

	successCount := 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		successCount++
		mockResponse := openweather_transport_http.OpenWeatherDTOResponse{
			Coord: openweather_transport_http.Coord{Lon: 37.6156, Lat: 55.7522},
			Name:  "Moscow",
			Cod:   200,
		}
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer testServer.Close()

	config := openweather_service.Config{
		APIBaseUrl:        testServer.URL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 10, // Низкий лимит для быстрого тестирования
		QueueSize:         5,
	}

	handler, mockLocationsService, rateLimiter, logger := setupTestHandler(t, config)
	defer rateLimiter.Close()

	// Мокируем CreateLocation для всех запросов
	mockLocationsService.On("CreateLocation", mock.Anything, mock.Anything).Return(domain.Location{ID: 1}, nil)

	// Отправляем больше запросов, чем разрешено лимитом
	requestsToSend := 20
	for i := 0; i < requestsToSend; i++ {
		requestBody := openweather_transport_http.GetOpenWeatherLocationRequest{
			CityName: "Moscow",
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-location", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		ctx := req.Context()
		ctx = core_logger.ToContext(ctx, logger)
		ctx = context.WithValue(ctx, core_http_middleware.UserIDContextKey, int64(123))

		// Добавляем короткий таймаут, чтобы не ждать долго в очереди
		ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.GetOpenWeatherLocation(rr, req)

		cancel()

		// Ожидаем либо успех, либо ошибку rate limit
		if rr.Code != http.StatusOK {
			t.Logf("Request %d: Rate limited (code=%d)", i+1, rr.Code)
		}
	}

	// Проверяем, что некоторые запросы были заблокированы rate limiter
	t.Logf("Successful requests: %d out of %d", successCount, requestsToSend)
	assert.Less(t, successCount, requestsToSend, "Some requests should have been rate limited")
}
