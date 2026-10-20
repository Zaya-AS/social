package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultHTTPPort = "8080"

type Config struct {
	HTTPPort string
}

func Load() (Config, error) {

	cfg := Config{
		HTTPPort: strings.TrimSpace(os.Getenv("HTTP_PORT")),
	}

	if cfg.HTTPPort == "" {
		cfg.HTTPPort = defaultHTTPPort
	}

	if err := validatePort(cfg.HTTPPort); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validatePort(value string) error {

	port, err := strconv.Atoi(value)

	if err != nil {
		return fmt.Errorf("HTTP_PORT must be a number: %w", err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("HTTP_PORT must be between 1 and 65535")
	}

	return nil
}
