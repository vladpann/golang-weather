package core_http_middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxBodySize_WithinLimit(t *testing.T) {
	maxBytes := int64(100)
	body := strings.Repeat("a", 50) // 50 bytes

	handler := MaxBodySize(maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(data))
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMaxBodySize_ExceedsLimit(t *testing.T) {
	maxBytes := int64(100)
	body := strings.Repeat("a", 150) // 150 bytes > 100

	handler := MaxBodySize(maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		// MaxBytesReader returns an error when limit is exceeded
		assert.Error(t, err)
		w.WriteHeader(http.StatusRequestEntityTooLarge)
	}))

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

func TestMaxBodySize_NoBody(t *testing.T) {
	maxBytes := int64(100)

	handler := MaxBodySize(maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMaxBodySize_ZeroLimit(t *testing.T) {
	maxBytes := int64(0) // 0 means no limit

	body := strings.Repeat("a", 1000)

	handler := MaxBodySize(maxBytes)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Len(t, data, 1000)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
