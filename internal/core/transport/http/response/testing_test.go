package core_http_response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	core_http_response "github.com/vladpann/backend/golang-weather/internal/core/transport/http/response"
)

func TestNewTestHandler(t *testing.T) {
	t.Run("создаёт handler без паники", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler := core_http_response.NewTestHandler(t, rr)

		assert.NotNil(t, handler)
	})

	t.Run("handler корректно обрабатывает ErrorResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler := core_http_response.NewTestHandler(t, rr)

		handler.ErrorResponse(core_errors.ErrNotFound, "user not found")

		assert.Equal(t, http.StatusNotFound, rr.Code)

		var response core_http_response.ErrorResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "not found", response.Error)
	})

	t.Run("handler корректно обрабатывает JSONResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler := core_http_response.NewTestHandler(t, rr)

		data := map[string]string{"status": "ok"}
		handler.JSONResponse(data, http.StatusOK)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	})
}

func TestNewTestHandlerWithLogs(t *testing.T) {
	t.Run("возвращает handler и наблюдаемые логи", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)

		assert.NotNil(t, handler)
		assert.NotNil(t, logs)
		assert.Equal(t, 0, logs.Len())
	})

	t.Run("логи записываются при ErrorResponse", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)

		testErr := errors.New("something went wrong")
		handler.ErrorResponse(testErr, "operation failed")

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		require.Equal(t, 1, logs.Len())

		entry := logs.All()[0]
		assert.Equal(t, "operation failed", entry.Message)
		assert.Contains(t, entry.ContextMap()["error"], "something went wrong")
	})

	t.Run("логи записываются для разных уровней", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)

		// ErrNotFound логируется на уровне Debug
		handler.ErrorResponse(core_errors.ErrNotFound, "not found")

		require.GreaterOrEqual(t, logs.Len(), 1)
		assert.Contains(t, logs.All()[0].Message, "not found")
	})
}

func TestNewTestContext(t *testing.T) {
	t.Run("создаёт контекст с логгером", func(t *testing.T) {
		ctx := core_http_response.NewTestContext(t)

		assert.NotNil(t, ctx)
	})

	t.Run("FromContext не паникует", func(t *testing.T) {
		ctx := core_http_response.NewTestContext(t)

		assert.NotPanics(t, func() {
			log := core_logger.FromContext(ctx)
			assert.NotNil(t, log)
		})
	})

	t.Run("контекст может использоваться в HTTP handler", func(t *testing.T) {
		// Simulate a real handler that uses FromContext
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := core_logger.FromContext(r.Context())
			log.Info("handling request")

			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := core_http_response.NewTestContext(t)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		assert.NotPanics(t, func() {
			handler.ServeHTTP(rr, req)
		})

		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

func TestNewTestContextWithLogs(t *testing.T) {
	t.Run("создаёт контекст и возвращает наблюдаемые логи", func(t *testing.T) {
		ctx, logs := core_http_response.NewTestContextWithLogs(t)

		assert.NotNil(t, ctx)
		assert.NotNil(t, logs)
		assert.Equal(t, 0, logs.Len())
	})

	t.Run("логи записываются из контекста", func(t *testing.T) {
		ctx, logs := core_http_response.NewTestContextWithLogs(t)

		log := core_logger.FromContext(ctx)
		log.Info("test message")

		require.Equal(t, 1, logs.Len())
		assert.Equal(t, "test message", logs.All()[0].Message)
	})

	t.Run("контекст используется в HTTP handler с проверкой логов", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log := core_logger.FromContext(r.Context())
			log.Info("request received")

			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx, logs := core_http_response.NewTestContextWithLogs(t)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.Equal(t, 1, logs.Len())
		assert.Equal(t, "request received", logs.All()[0].Message)
	})
}

func TestTestingHelpersIntegration(t *testing.T) {
	t.Run("полный сценарий с контекстом и handler", func(t *testing.T) {
		// 1. Создаём контекст для request
		ctx := core_http_response.NewTestContext(t)

		// 2. Создаём request с контекстом
		req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
		req = req.WithContext(ctx)

		// 3. Создаём ResponseRecorder и handler
		rr := httptest.NewRecorder()
		handler := core_http_response.NewTestHandler(t, rr)

		// 4. Используем handler
		handler.ErrorResponse(core_errors.ErrInvalidArgument, "validation failed")

		// 5. Проверяем результат
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response core_http_response.ErrorResponse
		err := json.NewDecoder(rr.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "invalid argument", response.Error)
		assert.Equal(t, "validation failed", response.Message)
	})
}

// TestExampleUsage демонстрирует типичные сценарии использования тестовых хелперов.
func TestExampleUsage(t *testing.T) {
	t.Run("тестирование простого handler", func(t *testing.T) {
		// Когда нужно просто проверить HTTP статус и тело ответа
		rr := httptest.NewRecorder()
		handler := core_http_response.NewTestHandler(t, rr)

		handler.JSONResponse(map[string]string{"result": "success"}, http.StatusOK)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("тестирование handler с проверкой логов", func(t *testing.T) {
		// Когда нужно проверить, что handler правильно логирует
		rr := httptest.NewRecorder()
		handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)

		handler.ErrorResponse(errors.New("db error"), "failed to save user")

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Equal(t, 1, logs.Len())
		assert.Contains(t, logs.All()[0].Message, "failed to save user")
	})

	t.Run("тестирование handler, использующего FromContext", func(t *testing.T) {
		// Когда handler вызывает core_logger.FromContext
		myHandler := func(rw http.ResponseWriter, r *http.Request) {
			log := core_logger.FromContext(r.Context())
			responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

			err := core_errors.ErrNotFound
			responseHandler.ErrorResponse(err, "resource not found")
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := core_http_response.NewTestContext(t)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		myHandler(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}
