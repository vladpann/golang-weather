package core_http_response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HTTPResponseHandler struct {
	rw http.ResponseWriter
}

func NewHTTPResponseHandler(
	rw http.ResponseWriter,
) *HTTPResponseHandler {
	return &HTTPResponseHandler{
		rw: rw,
	}
}

func (h *HTTPResponseHandler) JSONResponse(
	responseBody any,
	statusCode int,
) {
	h.rw.WriteHeader(statusCode)

	if err := json.NewEncoder(h.rw).Encode(responseBody); err != nil {
		fmt.Println("", err.Error())
	}
}

func (h *HTTPResponseHandler) NoContentResponse() {
	h.rw.WriteHeader(http.StatusNoContent)
}

func (h *HTTPResponseHandler) HTMLResponse(html []byte) {
	h.rw.WriteHeader(http.StatusOK)

	h.rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := h.rw.Write(html); err != nil {
		fmt.Println("", err.Error())
	}
}

func (h *HTTPResponseHandler) ErrorResponse(err error, msg string) {
	// var statusCode int
	statusCode := http.StatusInternalServerError

	h.errorResponse(
		statusCode,
		err,
		msg,
	)
}

func (h *HTTPResponseHandler) PanicResponse(p any, msg string) {
	statusCode := http.StatusInternalServerError
	err := fmt.Errorf("unexpected panic: %v", p)

	h.rw.WriteHeader(statusCode)

	h.errorResponse(
		statusCode,
		err,
		msg,
	)
}

func (h *HTTPResponseHandler) errorResponse(
	statusCode int,
	err error,
	msg string,
) {
	response := ErrorResponse{
		Error:   err.Error(),
		Message: msg,
	}

	h.JSONResponse(
		response,
		statusCode,
	)
}
