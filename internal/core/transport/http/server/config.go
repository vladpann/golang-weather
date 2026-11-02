package core_http_server

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr            string        `envconfig:"ADDR" required:"true"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"30s"`
	AllowedOrigins  []string      `envconfig:"ALLOWED_ORIGINS" required:"true"`

	// Timeouts for HTTP server to prevent resource exhaustion
	ReadHeaderTimeout time.Duration `envconfig:"READ_HEADER_TIMEOUT" default:"10s"`
	ReadTimeout       time.Duration `envconfig:"READ_TIMEOUT" default:"15s"`
	WriteTimeout      time.Duration `envconfig:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout       time.Duration `envconfig:"IDLE_TIMEOUT" default:"120s"`

	// Maximum request body size in bytes (before any processing)
	MaxBodyBytes int64 `envconfig:"MAX_BODY_BYTES" default:"1048576"` // 1MB default

	// Global request deadline covering DB, queue, upstream calls
	RequestDeadline time.Duration `envconfig:"REQUEST_DEADLINE" default:"25s"`

	// Maximum concurrent requests (0 = unlimited, not recommended for production)
	// Enables controlled overload rejection instead of resource exhaustion
	MaxConcurrentRequests int64 `envconfig:"MAX_CONCURRENT_REQUESTS" default:"500"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("HTTP", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get HTTP server config: %w", err)
		panic(err)
	}

	return config
}
