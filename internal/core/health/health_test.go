package core_health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func do(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func newMux(c *Checker) *http.ServeMux {
	mux := http.NewServeMux()
	c.Register(mux)
	return mux
}

func TestHealthzDoesNotTouchDB(t *testing.T) {
	c := New(func(context.Context) error { t.Fatal("ping must not be called"); return nil }, 0, nil)
	if rec := do(t, newMux(c), "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestReadyzOK(t *testing.T) {
	c := New(func(context.Context) error { return nil }, 0, nil)
	if rec := do(t, newMux(c), "/readyz"); rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestReadyzDBError(t *testing.T) {
	var got error
	c := New(func(context.Context) error { return errors.New("boom") }, 0, func(err error) { got = err })
	rec := do(t, newMux(c), "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rec.Code)
	}
	if got == nil {
		t.Fatal("onError was not called")
	}
	if rec.Body.String() == "boom" {
		t.Fatal("internal error leaked into response body")
	}
}

func TestReadyzTimeout(t *testing.T) {
	c := New(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, 20*time.Millisecond, nil)

	start := time.Now()
	rec := do(t, newMux(c), "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rec.Code)
	}
	if time.Since(start) > time.Second {
		t.Fatal("readyz did not respect timeout")
	}
}

func TestReadyzShuttingDown(t *testing.T) {
	c := New(func(context.Context) error { return nil }, 0, nil)
	c.SetShuttingDown()
	mux := newMux(c)
	if rec := do(t, mux, "/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz: got %d, want 503", rec.Code)
	}
	// liveness не должен падать при остановке
	if rec := do(t, mux, "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("healthz: got %d, want 200", rec.Code)
	}
}
