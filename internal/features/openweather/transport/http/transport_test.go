package openweather_transport_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/vladpann/backend/golang-weather/internal/core/domain"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"github.com/vladpann/backend/golang-weather/internal/features/openweather/ratelimiter"
	openweather_service "github.com/vladpann/backend/golang-weather/internal/features/openweather/service"
	openweather_transport_http "github.com/vladpann/backend/golang-weather/internal/features/openweather/transport/http"
)

// MockLocationsService нужен только для конструктора handler'а.
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

type testEnv struct {
	handler *openweather_transport_http.OpenWeatherHTTPHandler
	logger  *core_logger.Logger
}

// newTestEnv поднимает handler, который ходит в upstream-сервер по upstreamURL.
func newTestEnv(t *testing.T, upstreamURL string) *testEnv {
	t.Helper()

	config := openweather_service.Config{
		APIBaseUrl:        upstreamURL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	rl, err := ratelimiter.NewRateLimiter(ratelimiter.Config{
		RequestsPerMinute: config.RequestsPerMinute,
		QueueSize:         config.QueueSize,
	}, zapLogger)
	require.NoError(t, err)
	t.Cleanup(func() { rl.Close() })

	noopAuth := func(next http.Handler) http.Handler { return next }

	handler := openweather_transport_http.NewOpenWeatherHTTPHandler(
		new(MockLocationsService),
		config,
		noopAuth,
		rl,
	)

	return &testEnv{handler: handler, logger: &core_logger.Logger{Logger: zapLogger}}
}

func newTestEnvWithAuth(
	t *testing.T,
	upstreamURL string,
	auth func(http.Handler) http.Handler,
) *testEnv {
	t.Helper()

	config := openweather_service.Config{
		APIBaseUrl:        upstreamURL,
		APIKey:            "test-api-key",
		RequestsPerMinute: 60,
		QueueSize:         100,
	}

	zapLogger, err := zap.NewDevelopment()
	require.NoError(t, err)

	rl, err := ratelimiter.NewRateLimiter(ratelimiter.Config{
		RequestsPerMinute: config.RequestsPerMinute,
		QueueSize:         config.QueueSize,
	}, zapLogger)
	require.NoError(t, err)
	t.Cleanup(func() { rl.Close() })

	handler := openweather_transport_http.NewOpenWeatherHTTPHandler(
		new(MockLocationsService),
		config,
		auth,
		rl,
	)

	return &testEnv{handler: handler, logger: &core_logger.Logger{Logger: zapLogger}}
}

// do отправляет сырое тело в GetOpenWeather и возвращает recorder.
func (e *testEnv) do(t *testing.T, ctx context.Context, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/open-weather/get-weather", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(core_logger.ToContext(ctx, e.logger))

	rr := httptest.NewRecorder()
	e.handler.GetOpenWeather(rr, req)

	return rr
}

func (e *testEnv) doCoords(t *testing.T, ctx context.Context, lat, lon float64) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(openweather_transport_http.GetOpenWeatherRequest{
		Latitude:  lat,
		Longitude: lon,
	})
	require.NoError(t, err)

	return e.do(t, ctx, body)
}

func TestGetOpenWeather_Success(t *testing.T) {
	upstream := openweather_transport_http.OpenWeatherDTOResponse{
		Coord: openweather_transport_http.Coord{Lon: 37.6156, Lat: 55.7522},
		Weather: []openweather_transport_http.Weather{
			{ID: 800, Main: "Clear", Description: "clear sky", Icon: "01d"},
		},
		Main: openweather_transport_http.Main{
			Temp: 273.15, FeelsLike: 270.0, Pressure: 1013, Humidity: 80,
		},
		Sys:  openweather_transport_http.Sys{Country: "RU"},
		Name: "Moscow",
		Cod:  200,
	}
	upstreamBody, err := json.Marshal(upstream)
	require.NoError(t, err)

	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)

		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/data/2.5/weather", r.URL.Path)
		assert.Equal(t, "test-api-key", r.URL.Query().Get("appid"))

		lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
		assert.NoError(t, err)
		lon, err := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
		assert.NoError(t, err)
		assert.InDelta(t, 55.7522, lat, 1e-4)
		assert.InDelta(t, 37.6156, lon, 1e-4)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(upstreamBody)
	}))
	defer server.Close()

	env := newTestEnv(t, server.URL)

	rr := env.doCoords(t, context.Background(), 55.7522, 37.6156)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.EqualValues(t, 1, atomic.LoadInt32(&calls))

	var got openweather_transport_http.GetOpenWeatherResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))

	assert.Equal(t, "Moscow", got.Name)
	assert.Equal(t, 37.6156, got.Coord.Lon)
	assert.Equal(t, 55.7522, got.Coord.Lat)
	assert.Equal(t, 273.15, got.Main.Temp)
	assert.Equal(t, 80, got.Main.Humidity)
	assert.Equal(t, "RU", got.Sys.Country)
	require.Len(t, got.Weather, 1)
	assert.Equal(t, "Clear", got.Weather[0].Main)
}

func TestGetOpenWeather_UpstreamErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"bad request", http.StatusBadRequest, `{"cod":"400","message":"Invalid coordinates"}`},
		{"unauthorized", http.StatusUnauthorized, `{"cod":401,"message":"Invalid API key"}`},
		{"not found", http.StatusNotFound, `{"cod":"404","message":"city not found"}`},
		{"too many requests", http.StatusTooManyRequests, `{"cod":429,"message":"limit exceeded"}`},
		{"internal error", http.StatusInternalServerError, `{"message":"oops"}`},
		{"service unavailable", http.StatusServiceUnavailable, `{"message":"Service Unavailable"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			env := newTestEnv(t, server.URL)

			rr := env.doCoords(t, context.Background(), 55.7522, 37.6156)

			// Handler сейчас оборачивает любой не-200 в общую ошибку -> 500.
			assert.Equal(t, http.StatusInternalServerError, rr.Code)

			var resp map[string]any
			require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
			assert.Contains(t, resp, "error")
		})
	}
}

func TestGetOpenWeather_UpstreamInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"invalid json`))
	}))
	defer server.Close()

	env := newTestEnv(t, server.URL)

	rr := env.doCoords(t, context.Background(), 55.7522, 37.6156)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGetOpenWeather_UpstreamUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close() // порт закрыт -> Do вернёт ошибку

	env := newTestEnv(t, url)

	rr := env.doCoords(t, context.Background(), 55.7522, 37.6156)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestGetOpenWeather_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ``},
		{"malformed json", `{"latitude": `},
		{"missing fields", `{}`},
		{"missing longitude", `{"latitude": 55.7522}`},
		{"wrong types", `{"latitude": "abc", "longitude": "def"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
			}))
			defer server.Close()

			env := newTestEnv(t, server.URL)

			rr := env.do(t, context.Background(), []byte(tt.body))

			// Ожидаем 400 от DecodeAndValidateRequest (ErrInvalidArgument).
			// Если ваш ErrorResponse маппит иначе - поправьте ожидаемый код.
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.EqualValues(t, 0, atomic.LoadInt32(&calls), "upstream must not be called")
		})
	}
}

func TestGetOpenWeather_RateLimitRejected(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	env := newTestEnv(t, server.URL)

	// Отменённый контекст: Allow(ctx) не должен пропустить запрос.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rr := env.doCoords(t, ctx, 55.7522, 37.6156)

	assert.NotEqual(t, http.StatusOK, rr.Code)
	assert.EqualValues(t, 0, atomic.LoadInt32(&calls), "upstream must not be called when rate limited")
}

func TestOpenWeatherHTTPHandler_Routes(t *testing.T) {
	var authCalls int32
	markAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&authCalls, 1)
			next.ServeHTTP(w, r)
		})
	}

	env := newTestEnvWithAuth(t, "http://unused", markAuth)
	routes := env.handler.Routes()
	require.Len(t, routes, 2)

	t.Run("search-location", func(t *testing.T) {
		route := routes[0]
		assert.Equal(t, http.MethodGet, route.Method)
		assert.Equal(t, "/open-weather/search-location", route.Path)
		assert.NotNil(t, route.Handler)
		require.Len(t, route.Middlewares, 1)
	})

	t.Run("get-weather", func(t *testing.T) {
		route := routes[1]
		assert.Equal(t, http.MethodPost, route.Method)
		assert.Equal(t, "/open-weather/get-weather", route.Path)
		assert.NotNil(t, route.Handler)
		require.Len(t, route.Middlewares, 1)
	})

	t.Run("в маршрутах стоит именно наш auth middleware", func(t *testing.T) {
		for _, route := range routes {
			before := atomic.LoadInt32(&authCalls)

			// Применяем middleware маршрута к заглушке и вызываем
			wrapped := route.Middlewares[0](http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) },
			))
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, httptest.NewRequest(route.Method, route.Path, nil))

			assert.Equal(t, http.StatusNoContent, rr.Code)
			assert.EqualValues(t, before+1, atomic.LoadInt32(&authCalls), route.Path)
		}
	})

	t.Run("get-weather привязан к GetOpenWeather", func(t *testing.T) {
		// Пустое тело даёт 400 от DecodeAndValidateRequest, как в TestGetOpenWeather_InvalidRequest
		req := httptest.NewRequest(http.MethodPost, "/open-weather/get-weather", bytes.NewReader(nil))
		req = req.WithContext(core_logger.ToContext(req.Context(), env.logger))
		rr := httptest.NewRecorder()

		routes[1].Handler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

// TestSearchLocation_Success проверяет успешный поиск локации
func TestSearchLocation_Success(t *testing.T) {
	upstreamResponse := []openweather_transport_http.GeocodingResponse{
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

	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем правильность URL
		assert.Equal(t, "/geo/1.0/direct", r.URL.Path)

		// Проверяем query параметры
		query := r.URL.Query()
		assert.Equal(t, "Moscow", query.Get("q"))
		assert.Equal(t, "5", query.Get("limit"))
		assert.Equal(t, "test-api-key", query.Get("appid"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(upstreamResponse)
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=Moscow", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []openweather_transport_http.GeocodingResponse
	err := json.NewDecoder(rr.Body).Decode(&result)
	require.NoError(t, err)

	assert.Len(t, result, 2)
	assert.Equal(t, "Moscow", result[0].Name)
	assert.Equal(t, 55.7522, result[0].Lat)
	assert.Equal(t, 37.6156, result[0].Lon)
	assert.Equal(t, "RU", result[0].Country)
}

// TestSearchLocation_MissingCityName проверяет обработку отсутствующего параметра city_name
func TestSearchLocation_MissingCityName(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream не должен вызываться при отсутствии city_name")
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	// ErrorResponse возвращает StatusInternalServerError
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// TestSearchLocation_UpstreamErrors проверяет обработку различных ошибок от upstream
func TestSearchLocation_UpstreamErrors(t *testing.T) {
	testCases := []struct {
		name           string
		upstreamStatus int
		expectedStatus int
	}{
		{
			name:           "bad_request",
			upstreamStatus: http.StatusBadRequest,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unauthorized",
			upstreamStatus: http.StatusUnauthorized,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "not_found",
			upstreamStatus: http.StatusNotFound,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "too_many_requests",
			upstreamStatus: http.StatusTooManyRequests,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "internal_error",
			upstreamStatus: http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "service_unavailable",
			upstreamStatus: http.StatusServiceUnavailable,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.upstreamStatus)
			}))
			defer upstreamServer.Close()

			env := newTestEnv(t, upstreamServer.URL)
			ctx := context.Background()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=Test", nil)
			req = req.WithContext(core_logger.ToContext(ctx, env.logger))
			rr := httptest.NewRecorder()

			env.handler.SearchLocation(rr, req)

			assert.Equal(t, tc.expectedStatus, rr.Code)
		})
	}
}

// TestSearchLocation_UpstreamInvalidJSON проверяет обработку невалидного JSON от upstream
func TestSearchLocation_UpstreamInvalidJSON(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=Test", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// TestSearchLocation_UpstreamUnreachable проверяет обработку недоступного upstream
func TestSearchLocation_UpstreamUnreachable(t *testing.T) {
	env := newTestEnv(t, "http://127.0.0.1:64212") // несуществующий адрес
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=Test", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// TestSearchLocation_RateLimitRejected проверяет обработку превышения rate limit
func TestSearchLocation_RateLimitRejected(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream не должен вызываться при превышении rate limit")
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)

	// Отменяем контекст, чтобы симулировать ошибку rate limiter (context.Canceled)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=Test", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	// context.Canceled возвращает StatusRequestTimeout (408)
	assert.Equal(t, http.StatusRequestTimeout, rr.Code)
}

// TestSearchLocation_SpecialCharacters проверяет обработку спецсимволов в названии города
func TestSearchLocation_SpecialCharacters(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		cityName := query.Get("q")

		// Проверяем, что спецсимволы правильно escaped
		assert.Equal(t, "San Francisco", cityName)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]openweather_transport_http.GeocodingResponse{
			{Name: "San Francisco", Lat: 37.7749, Lon: -122.4194, Country: "US"},
		})
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)
	ctx := context.Background()

	// URL encoding будет выполнен браузером/клиентом
	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=San+Francisco", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

// TestSearchLocation_EmptyResults проверяет обработку пустого результата от API
func TestSearchLocation_EmptyResults(t *testing.T) {
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]openweather_transport_http.GeocodingResponse{})
	}))
	defer upstreamServer.Close()

	env := newTestEnv(t, upstreamServer.URL)
	ctx := context.Background()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/open-weather/search-location?city_name=NonExistentCity12345", nil)
	req = req.WithContext(core_logger.ToContext(ctx, env.logger))
	rr := httptest.NewRecorder()

	env.handler.SearchLocation(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var result []openweather_transport_http.GeocodingResponse
	err := json.NewDecoder(rr.Body).Decode(&result)
	require.NoError(t, err)
	assert.Len(t, result, 0)
}
