package ratelimiter

import "errors"

var (
	// ErrQueueFull is returned when the rate limiter queue is at capacity
	ErrQueueFull = errors.New("rate limit queue is full")
	// ErrClosed is returned by Allow when the rate limiter has been closed
	// (or is closed while the request is waiting in the queue).
	ErrClosed = errors.New("rate limiter is closed")
)
