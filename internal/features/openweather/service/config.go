package openweather_service

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	APIBaseUrl        string `envconfig:"API_BASE_URL" required:"true"`
	APIKey            string `envconfig:"API_KEY" required:"true"`
	RequestsPerMinute int    `envconfig:"REQUESTS_PER_MINUTE" default:"60"`
	QueueSize         int    `envconfig:"QUEUE_SIZE" default:"100"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("OPENWEATHER", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get OpenWeather config: %w", err)
		panic(err)
	}

	return config
}
