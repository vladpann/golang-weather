package sessions_worker

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Cleanup_Inverval time.Duration `envconfig:"CLEANUP_INTERVAL" default:"10h"`
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
