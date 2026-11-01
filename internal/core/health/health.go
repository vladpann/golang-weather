package core_health

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

const defaultTimeout = 2 * time.Second

type PingFunc func(ctx context.Context) error

type Checker struct {
	ping    PingFunc
	timeout time.Duration
	onError func(error)
	closing atomic.Bool
}

func New(ping PingFunc, timeout time.Duration, onError func(error)) *Checker {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Checker{ping: ping, timeout: timeout, onError: onError}
}

func (c *Checker) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", c.Healthz)
	mux.HandleFunc("GET /readyz", c.Readyz)
}

func (c *Checker) SetShuttingDown() { c.closing.Store(true) }

func (c *Checker) Healthz(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, "ok")
}

func (c *Checker) Readyz(w http.ResponseWriter, r *http.Request) {
	if c.closing.Load() {
		write(w, http.StatusServiceUnavailable, "shutting down")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), c.timeout)
	defer cancel()

	if err := c.ping(ctx); err != nil {
		if c.onError != nil {
			c.onError(err)
		}
		write(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	write(w, http.StatusOK, "ok")
}

func write(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}
