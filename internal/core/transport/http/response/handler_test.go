package core_http_response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

// newTestLogger создает логгер для тестов
func newTestLogger() *core_logger.Logger {
	zapLogger := zap.NewNop()
	return &core_logger.Logger{
		Logger: zapLogger,
	}
}

// newObservedLogger возвращает логгер и доступ к записанным логам
func newObservedLogger() (*core_logger.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return &core_logger.Logger{
		Logger: zap.New(core),
	}, logs
}

// TestErrorResponse_WithError: внутренняя ошибка логируется, но не уходит клиенту
func TestErrorResponse_WithError(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	log, logs := newObservedLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	testErr := errors.New("test error")
	testMsg := "Something went wrong"

	// Act
	handler.ErrorResponse(testErr, testMsg)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response core_http_response.ErrorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))

	// Клиент не должен видеть текст внутренней ошибки
	assert.Empty(t, response.Error)
	assert.NotContains(t, rr.Body.String(), "test error")
	assert.Equal(t, testMsg, response.Message)

	// Но в лог ошибка должна попасть
	require.GreaterOrEqual(t, logs.Len(), 1)
	assert.Contains(t, logs.All()[0].ContextMap()["error"], "test error")
}

// TestErrorResponse_WithNilError тестирует ErrorResponse с nil ошибкой (регрессионный тест для бага)
func TestErrorResponse_WithNilError(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	log := newTestLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	testMsg := "User not authenticated"

	// Act - не должно вызвать панику
	handler.ErrorResponse(nil, testMsg)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response core_http_response.ErrorResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	// Когда err == nil, поле Error должно быть пустой строкой
	assert.Equal(t, "", response.Error)
	assert.Equal(t, testMsg, response.Message)
}

// TestJSONResponse_Success тестирует успешный JSON ответ
func TestJSONResponse_Success(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	log := newTestLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	type TestResponse struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	testData := TestResponse{
		ID:   123,
		Name: "Test",
	}

	// Act
	handler.JSONResponse(testData, http.StatusOK)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response TestResponse
	err := json.NewDecoder(rr.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, testData.ID, response.ID)
	assert.Equal(t, testData.Name, response.Name)
}

// TestJSONResponse_WithDifferentStatusCodes тестирует различные HTTP статусы
func TestJSONResponse_WithDifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"Bad Request", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"Not Found", http.StatusNotFound},
		{"Internal Server Error", http.StatusInternalServerError},
		{"Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			rr := httptest.NewRecorder()
			log := newTestLogger()
			handler := core_http_response.NewHTTPResponseHandler(log, rr)

			data := map[string]string{"status": tc.name}

			// Act
			handler.JSONResponse(data, tc.statusCode)

			// Assert
			assert.Equal(t, tc.statusCode, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
		})
	}
}

// TestNoContentResponse тестирует NoContentResponse
func TestNoContentResponse(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	log := newTestLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	// Act
	handler.NoContentResponse()

	// Assert
	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Empty(t, rr.Body.String())
}

// TestHTMLResponse тестирует HTMLResponse
func TestHTMLResponse(t *testing.T) {
	// Arrange
	rr := httptest.NewRecorder()
	log := newTestLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	htmlContent := []byte("<html><body>Test</body></html>")

	// Act
	handler.HTMLResponse(htmlContent)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/html; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Equal(t, string(htmlContent), rr.Body.String())
}

func TestPanicResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	log, logs := newObservedLogger()
	handler := core_http_response.NewHTTPResponseHandler(log, rr)

	panicValue := "something went terribly wrong"
	testMsg := "Unexpected panic occurred"

	handler.PanicResponse(panicValue, testMsg)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var response core_http_response.ErrorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&response))

	assert.Empty(t, response.Error)
	assert.NotContains(t, rr.Body.String(), panicValue)
	assert.Equal(t, testMsg, response.Message)

	require.Equal(t, 1, logs.Len())
	entry := logs.All()[0]
	assert.Equal(t, testMsg, entry.Message)
	assert.Equal(t, panicValue, entry.ContextMap()["panic"])
	assert.NotEmpty(t, entry.ContextMap()["stack"])
}
