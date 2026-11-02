package openweather_client

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	APIBaseURL        string        `envconfig:"API_BASE_URL" required:"true"`
	APIKey            string        `envconfig:"API_KEY" required:"true"`
	Timeout           time.Duration `envconfig:"TIMEOUT" default:"10s"`
	MaxResponseSize   int64         `envconfig:"MAX_RESPONSE_SIZE" default:"1048576"` // 1MB
	MaxRetries        int           `envconfig:"MAX_RETRIES" default:"2"`
	RetryDelay        time.Duration `envconfig:"RETRY_DELAY" default:"100ms"`
	CacheTTL          time.Duration `envconfig:"CACHE_TTL" default:"5m"`
	CacheCleanupInt   time.Duration `envconfig:"CACHE_CLEANUP_INTERVAL" default:"10m"`
	RequestsPerMinute int           `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
	QueueSize         int           `envconfig:"QUEUE_SIZE" default:"100"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("OPENWEATHER", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	if err := config.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return config, nil
}

func (c Config) Validate() error {
	if c.APIBaseURL == "" {
		return fmt.Errorf("API_BASE_URL is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("API_KEY is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be positive")
	}
	if c.MaxResponseSize <= 0 {
		return fmt.Errorf("MAX_RESPONSE_SIZE must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("MAX_RETRIES must be non-negative")
	}
	if c.RequestsPerMinute <= 0 {
		return fmt.Errorf("REQUESTS_PER_MINUTE must be positive")
	}
	if c.QueueSize <= 0 {
		return fmt.Errorf("QUEUE_SIZE must be positive")
	}
	return nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get OpenWeather client config: %w", err)
		panic(err)
	}

	return config
}
