package ratelimiter

import "fmt"

// Config holds rate limiter configuration
type Config struct {
	RequestsPerMinute int
	QueueSize         int
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.RequestsPerMinute <= 0 {
		return fmt.Errorf("requests per minute must be greater than 0, got %d", c.RequestsPerMinute)
	}
	if c.QueueSize <= 0 {
		return fmt.Errorf("queue size must be greater than 0, got %d", c.QueueSize)
	}
	return nil
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() Config {
	return Config{
		RequestsPerMinute: 60,
		QueueSize:         100,
	}
}
