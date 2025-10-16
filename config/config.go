package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port            string
	Environment     string
	LogLevel        string
	ApplicationAPIs []string
	BalancerType    string

	// Health Check Config
	HealthCheckInterval time.Duration

	// Circuit Breaker Config
	MaxFailures    int
	CircuitTimeout time.Duration
	ResetTimeout   time.Duration

	// Retry Config
	MaxRetries int
	RetryDelay time.Duration
}

func Load() (*Config, error) {
	config := &Config{
		Port:            getEnv("PORT", "3000"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ApplicationAPIs: getApplicationAPIs(),
		BalancerType:    getEnv("BALANCER_TYPE", "round-robin"),

		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", "5s"),

		MaxFailures:    getEnvInt("MAX_FAILURES", 5),
		CircuitTimeout: getEnvDuration("CIRCUIT_TIMEOUT", "30s"),
		ResetTimeout:   getEnvDuration("RESET_TIMEOUT", "60s"),

		MaxRetries: getEnvInt("MAX_RETRIES", 3),
		RetryDelay: getEnvDuration("RETRY_DELAY", "100ms"),
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

func (c *Config) Validate() error {
	if c.Port == "" {
		return errors.New("port cannot be empty")
	}
	if len(c.ApplicationAPIs) == 0 {
		return errors.New("at least one application API must be configured")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRaw(key string) string {
	return os.Getenv(key)
}

func getApplicationAPIs() []string {
	var apis []string

	for i := 1; i <= 10; i++ {
		if api := os.Getenv("API_" + strconv.Itoa(i)); api != "" {
			apis = append(apis, api)
		}
	}

	if len(apis) == 0 {
		apis = []string{
			"http://localhost:8080",
			"http://localhost:8081",
			"http://localhost:8082",
		}
	}

	return apis
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}
	return time.Duration(0)
}
