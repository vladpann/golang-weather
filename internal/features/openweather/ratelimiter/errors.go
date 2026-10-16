package ratelimiter

import "errors"

var (
	// ErrQueueFull is returned when the rate limiter queue is at capacity
	ErrQueueFull = errors.New("rate limit queue is full")
)
