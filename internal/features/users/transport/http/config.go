package users_transport_http

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Secure bool `envconfig:"SECURE" default:"true"`
}

func NewConfig() (Config, error) {
	var config Config

	if err := envconfig.Process("COOKIE", &config); err != nil {
		return Config{}, fmt.Errorf("process envconfig: %w", err)
	}

	return config, nil
}

func NewConfigMust() Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get Userss config: %w", err)
		panic(err)
	}

	return config
}
