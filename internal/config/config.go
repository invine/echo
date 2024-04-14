package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func NewConfig() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("parsing environment variables: %w", err)
	}
	return &config, nil
}
