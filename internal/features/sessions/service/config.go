package sessions_service

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	TTL time.Duration `envconfig:"TTL" default:"24h"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("SESSION", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get Sessions config: %w", err)
		panic(err)
	}

	return config
}
