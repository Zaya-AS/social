package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort              string
	DatabaseURL           string
	RedisAddr             string
	RedisDB               int
	RedisCacheTTL         time.Duration
	QueueCapacity         int
	WorkerCount           int
	LikeExecutionTimeout  time.Duration
	ConnectTimeout        time.Duration
	HTTPShutdownTimeout   time.Duration
	WorkerShutdownTimeout time.Duration
}

func Load() (Config, error) {

	cfg := Config{
		HTTPPort:    strings.TrimSpace(os.Getenv("HTTP_PORT")),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisAddr:   strings.TrimSpace(os.Getenv("REDIS_ADDR")),
	}

	if cfg.HTTPPort == "" {
		return Config{}, fmt.Errorf("HTTP_PORT is required")
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR is required")
	}

	if err := validatePort(cfg.HTTPPort); err != nil {
		return Config{}, err
	}

	redisDB, err := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB")))
	if err != nil {
		return Config{}, fmt.Errorf("REDIS_DB must be an integer: %w", err)
	}
	if redisDB < 0 {
		return Config{}, fmt.Errorf("REDIS_DB must be non-negative")
	}
	cfg.RedisDB = redisDB

	queueCap, err := strconv.Atoi(strings.TrimSpace(os.Getenv("QUEUE_CAPACITY")))
	if err != nil {
		return Config{}, fmt.Errorf("QUEUE_CAPACITY must be an integer: %w", err)
	}
	if queueCap <= 0 {
		return Config{}, fmt.Errorf("QUEUE_CAPACITY must be greater than zero")
	}
	cfg.QueueCapacity = queueCap

	worker, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORKER_COUNT")))
	if err != nil {
		return Config{}, fmt.Errorf("WORKER_COUNT must be an integer: %w", err)
	}
	if worker <= 0 {
		return Config{}, fmt.Errorf("WORKER_COUNT must be greater than zero")
	}
	cfg.WorkerCount = worker

	redisTTL, err := time.ParseDuration(strings.TrimSpace(os.Getenv("REDIS_CACHE_TTL")))
	if err != nil {
		return Config{}, fmt.Errorf(
			"REDIS_CACHE_TTL must be a valid duration: %w", err,
		)
	}

	if redisTTL < 5*time.Second || redisTTL > 30*time.Second {
		return Config{}, fmt.Errorf(
			"REDIS_CACHE_TTL must be between 5s and 30s",
		)
	}
	cfg.RedisCacheTTL = redisTTL

	executionTimeout, err := time.ParseDuration(
		strings.TrimSpace(os.Getenv("LIKE_EXECUTION_TIMEOUT")),
	)
	if err != nil {
		return Config{}, fmt.Errorf(
			"LIKE_EXECUTION_TIMEOUT must be a valid duration: %w", err,
		)
	}
	if executionTimeout <= 0 {
		return Config{}, fmt.Errorf(
			"LIKE_EXECUTION_TIMEOUT must be greater than zero",
		)
	}

	cfg.LikeExecutionTimeout = executionTimeout

	connectTimeout, err := time.ParseDuration(
		strings.TrimSpace(os.Getenv("CONNECT_TIMEOUT")),
	)
	if err != nil {
		return Config{}, fmt.Errorf("CONNECT_TIMEOUT must be a valid duration: %w", err)
	}
	if connectTimeout <= 0 {
		return Config{}, fmt.Errorf("CONNECT_TIMEOUT must be greater than zero")
	}
	cfg.ConnectTimeout = connectTimeout

	httpShutdownTimeout, err := time.ParseDuration(
		strings.TrimSpace(os.Getenv("HTTP_SHUTDOWN_TIMEOUT")),
	)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_SHUTDOWN_TIMEOUT must be a valid duration: %w", err)
	}
	if httpShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("HTTP_SHUTDOWN_TIMEOUT must be greater than zero")
	}
	cfg.HTTPShutdownTimeout = httpShutdownTimeout

	workerShutdownTimeout, err := time.ParseDuration(
		strings.TrimSpace(os.Getenv("WORKER_SHUTDOWN_TIMEOUT")),
	)
	if err != nil {
		return Config{}, fmt.Errorf("WORKER_SHUTDOWN_TIMEOUT must be a valid duration: %w", err)
	}
	if workerShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("WORKER_SHUTDOWN_TIMEOUT must be greater than zero")
	}
	cfg.WorkerShutdownTimeout = workerShutdownTimeout

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
