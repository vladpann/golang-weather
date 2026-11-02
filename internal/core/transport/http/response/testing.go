package core_http_response

import (
	"context"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
)

// NewTestHandler создаёт HTTPResponseHandler для тестов с Nop-логгером.
//
// Используется для тестирования HTTP handlers без реальных логов.
//
// Пример:
//
//	func TestMyHandler(t *testing.T) {
//	    rr := httptest.NewRecorder()
//	    handler := core_http_response.NewTestHandler(t, rr)
//
//	    handler.ErrorResponse(core_errors.ErrNotFound, "user not found")
//
//	    assert.Equal(t, http.StatusNotFound, rr.Code)
//	}
func NewTestHandler(t *testing.T, rw *httptest.ResponseRecorder) *HTTPResponseHandler {
	t.Helper()

	zapLogger := zap.NewNop()
	logger := &core_logger.Logger{Logger: zapLogger}

	return NewHTTPResponseHandler(logger, rw)
}

// NewTestHandlerWithLogs создаёт HTTPResponseHandler для тестов с наблюдаемыми логами.
//
// Возвращает handler и ObservedLogs для проверки того, что было залогировано.
//
// Пример:
//
//	func TestMyHandlerLogs(t *testing.T) {
//	    rr := httptest.NewRecorder()
//	    handler, logs := core_http_response.NewTestHandlerWithLogs(t, rr)
//
//	    handler.ErrorResponse(errors.New("internal error"), "something failed")
//
//	    assert.Equal(t, http.StatusInternalServerError, rr.Code)
//	    assert.Equal(t, 1, logs.Len())
//	    assert.Contains(t, logs.All()[0].Message, "something failed")
//	}
func NewTestHandlerWithLogs(t *testing.T, rw *httptest.ResponseRecorder) (*HTTPResponseHandler, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	logger := &core_logger.Logger{Logger: zap.New(core)}

	return NewHTTPResponseHandler(logger, rw), logs
}

// NewTestContext создаёт контекст с тестовым логгером для использования в HTTP handlers.
//
// Необходимо, так как core_logger.FromContext паникует, если логгер не установлен.
//
// Пример:
//
//	func TestHTTPHandler(t *testing.T) {
//	    req := httptest.NewRequest("GET", "/users", nil)
//	    ctx := core_http_response.NewTestContext(t)
//	    req = req.WithContext(ctx)
//
//	    rr := httptest.NewRecorder()
//	    handler.ServeHTTP(rr, req)
//
//	    assert.Equal(t, http.StatusOK, rr.Code)
//	}
func NewTestContext(t *testing.T) context.Context {
	t.Helper()

	zapLogger := zap.NewNop()
	logger := &core_logger.Logger{Logger: zapLogger}

	return core_logger.ToContext(context.Background(), logger)
}

// NewTestContextWithLogs создаёт контекст с наблюдаемым логгером.
//
// Возвращает контекст и ObservedLogs для проверки логирования.
//
// Пример:
//
//	func TestHTTPHandlerWithLogs(t *testing.T) {
//	    req := httptest.NewRequest("POST", "/users", body)
//	    ctx, logs := core_http_response.NewTestContextWithLogs(t)
//	    req = req.WithContext(ctx)
//
//	    rr := httptest.NewRecorder()
//	    handler.ServeHTTP(rr, req)
//
//	    assert.Equal(t, 1, logs.Len())
//	    assert.Equal(t, "user created", logs.All()[0].Message)
//	}
func NewTestContextWithLogs(t *testing.T) (context.Context, *observer.ObservedLogs) {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	logger := &core_logger.Logger{Logger: zap.New(core)}

	return core_logger.ToContext(context.Background(), logger), logs
}
