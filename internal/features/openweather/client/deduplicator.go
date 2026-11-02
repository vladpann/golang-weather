package openweather_client

import (
	"context"
	"sync"
)

// inflightRequest represents an in-flight request
type inflightRequest struct {
	wg   sync.WaitGroup
	data interface{}
	err  error
}

// Deduplicator prevents duplicate concurrent requests for the same key
type Deduplicator struct {
	mu      sync.Mutex
	inflight map[string]*inflightRequest
}

// NewDeduplicator creates a new request deduplicator
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		inflight: make(map[string]*inflightRequest),
	}
}

// Do executes the function or waits for an in-flight request with the same key
func (d *Deduplicator) Do(ctx context.Context, key string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	d.mu.Lock()

	// Check if there's already an in-flight request
	if req, found := d.inflight[key]; found {
		d.mu.Unlock()

		// Wait for the in-flight request to complete or context to be cancelled
		done := make(chan struct{})
		go func() {
			req.wg.Wait()
			close(done)
		}()

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-done:
			return req.data, req.err
		}
	}

	// Create new in-flight request
	req := &inflightRequest{}
	req.wg.Add(1)
	d.inflight[key] = req
	d.mu.Unlock()

	// Execute the function
	data, err := fn(ctx)

	// Store results and signal completion
	req.data = data
	req.err = err
	req.wg.Done()

	// Remove from in-flight map
	d.mu.Lock()
	delete(d.inflight, key)
	d.mu.Unlock()

	return data, err
}
