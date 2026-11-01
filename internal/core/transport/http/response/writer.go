package core_http_response

import "net/http"

const StatusCodeUninitialized = -1

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     StatusCodeUninitialized,
	}
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	if rw.statusCode == StatusCodeUninitialized {
		rw.statusCode = statusCode
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == StatusCodeUninitialized {
		rw.statusCode = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *ResponseWriter) GetStatusCode() int {
	if rw.statusCode == StatusCodeUninitialized {
		return http.StatusOK
	}
	return rw.statusCode
}
