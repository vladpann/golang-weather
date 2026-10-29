package core_http_response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	core_errors "github.com/vladpann/backend/golang-weather/internal/core/errors"
	core_logger "github.com/vladpann/backend/golang-weather/internal/core/logger"
	"go.uber.org/zap"
)

type HTTPResponseHandler struct {
	log *core_logger.Logger
	rw  http.ResponseWriter
}

func NewHTTPResponseHandler(
	log *core_logger.Logger,
	rw http.ResponseWriter,
) *HTTPResponseHandler {
	return &HTTPResponseHandler{log: log, rw: rw}
}

func (h *HTTPResponseHandler) JSONResponse(responseBody any, statusCode int) {
	h.rw.Header().Set("Content-Type", "application/json")
	h.rw.WriteHeader(statusCode)

	if err := json.NewEncoder(h.rw).Encode(responseBody); err != nil {
		h.log.Error("write HTTP response", zap.Error(err))
	}
}

func (h *HTTPResponseHandler) NoContentResponse() {
	h.rw.WriteHeader(http.StatusNoContent)
}

func (h *HTTPResponseHandler) HTMLResponse(html []byte) {
	h.rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.rw.WriteHeader(http.StatusOK)

	if _, err := h.rw.Write(html); err != nil {
		h.log.Error("write HTML HTTP response", zap.Error(err))
	}
}

func (h *HTTPResponseHandler) ErrorResponse(err error, msg string) {
	var (
		statusCode int
		logFunc    func(string, ...zap.Field)
		publicErr  string
	)

	switch {
	case errors.Is(err, core_errors.ErrInvalidArgument):
		statusCode, logFunc, publicErr = http.StatusBadRequest, h.log.Warn, core_errors.ErrInvalidArgument.Error()

	case errors.Is(err, core_errors.ErrNotFound):
		statusCode, logFunc, publicErr = http.StatusNotFound, h.log.Debug, core_errors.ErrNotFound.Error()

	case errors.Is(err, core_errors.ErrConflict):
		statusCode, logFunc, publicErr = http.StatusConflict, h.log.Warn, core_errors.ErrConflict.Error()

	case errors.Is(err, core_errors.ErrLoginAlreadyExists):
		statusCode, logFunc, publicErr = http.StatusConflict, h.log.Warn, core_errors.ErrLoginAlreadyExists.Error()

	case errors.Is(err, core_errors.ErrLoginRequired):
		statusCode, logFunc, publicErr = http.StatusBadRequest, h.log.Warn, core_errors.ErrLoginRequired.Error()

	case errors.Is(err, core_errors.ErrPasswordRequired):
		statusCode, logFunc, publicErr = http.StatusBadRequest, h.log.Warn, core_errors.ErrPasswordRequired.Error()

	case errors.Is(err, core_errors.ErrPasswordRequired):
		statusCode, logFunc, publicErr = http.StatusBadRequest, h.log.Warn, core_errors.ErrPasswordRequired.Error()

	case errors.Is(err, core_errors.ErrPasswordDoNotMatch):
		statusCode, logFunc, publicErr = http.StatusBadRequest, h.log.Warn, core_errors.ErrPasswordDoNotMatch.Error()

	case errors.Is(err, core_errors.ErrInvalidCredentials):
		statusCode, logFunc, publicErr = http.StatusUnauthorized, h.log.Warn, core_errors.ErrInvalidCredentials.Error()

	default:
		statusCode, logFunc, publicErr = http.StatusInternalServerError, h.log.Error, ""
	}

	logFunc(msg, zap.Error(err))

	h.errorResponse(statusCode, publicErr, msg)
}

func (h *HTTPResponseHandler) PanicResponse(p any, msg string) {
	h.log.Error(
		msg,
		zap.String("panic", fmt.Sprint(p)),
		zap.ByteString("stack", debug.Stack()),
	)

	h.errorResponse(http.StatusInternalServerError, "", msg)
}

func (h *HTTPResponseHandler) errorResponse(
	statusCode int,
	publicErr string,
	msg string,
) {
	h.JSONResponse(
		ErrorResponse{
			Error:   publicErr,
			Message: msg,
		},
		statusCode,
	)
}
