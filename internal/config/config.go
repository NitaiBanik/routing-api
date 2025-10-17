package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	SlowThreshold  time.Duration
	MaxSlowCount   int

	// Retry Config
	MaxRetries int
	RetryDelay time.Duration

	// HTTP Client Timeouts
	RequestTimeout  time.Duration
	ConnectTimeout  time.Duration
	ResponseTimeout time.Duration
}

func Load() (*Config, error) {
	return LoadWithDefaults(true)
}

func LoadWithDefaults(useDefaults bool) (*Config, error) {
	var port string
	var maxFailures, maxRetries int
	var err error

	if useDefaults {
		port = getEnv("PORT", "3000")
		maxFailures = getEnvInt("MAX_FAILURES", 5)
		maxRetries = getEnvInt("MAX_RETRIES", 3)
	} else {
		port = getEnvRaw("PORT")
		maxFailures, err = getEnvIntRaw("MAX_FAILURES")
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_FAILURES: %w", err)
		}
		maxRetries, err = getEnvIntRaw("MAX_RETRIES")
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_RETRIES: %w", err)
		}
	}

	config := &Config{
		Port:            port,
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ApplicationAPIs: getApplicationAPIsWithDefaults(useDefaults),
		BalancerType:    getEnv("BALANCER_TYPE", "round-robin"),

		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", "5s"),

		MaxFailures:    maxFailures,
		CircuitTimeout: getEnvDuration("CIRCUIT_TIMEOUT", "30s"),
		ResetTimeout:   getEnvDuration("RESET_TIMEOUT", "60s"),
		SlowThreshold:  getEnvDuration("SLOW_THRESHOLD", "5s"),
		MaxSlowCount:   getEnvInt("MAX_SLOW_COUNT", 3),

		MaxRetries: maxRetries,
		RetryDelay: getEnvDuration("RETRY_DELAY", "100ms"),

		RequestTimeout:  getEnvDuration("REQUEST_TIMEOUT", "30s"),
		ConnectTimeout:  getEnvDuration("CONNECT_TIMEOUT", "5s"),
		ResponseTimeout: getEnvDuration("RESPONSE_TIMEOUT", "25s"),
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
	return getApplicationAPIsWithDefaults(true)
}

func getApplicationAPIsWithDefaults(useDefaults bool) []string {
	var apis []string

	// Check APPLICATION_APIS environment variable first
	if apisEnv := os.Getenv("APPLICATION_APIS"); apisEnv != "" {
		// Split by comma and trim spaces
		parts := strings.Split(apisEnv, ",")
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				apis = append(apis, trimmed)
			}
		}
		return apis
	}

	// Fallback to individual API_1, API_2, etc.
	for i := 1; i <= 10; i++ {
		if api := os.Getenv("API_" + strconv.Itoa(i)); api != "" {
			apis = append(apis, api)
		}
	}

	// Only return defaults if useDefaults is true and no APIs are configured
	if useDefaults && len(apis) == 0 {
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

func getEnvIntRaw(key string) (int, error) {
	if value := os.Getenv(key); value != "" {
		return strconv.Atoi(value)
	}
	return 0, errors.New("environment variable not set")
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
