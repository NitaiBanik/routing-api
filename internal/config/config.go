package config

import (
	"bufio"
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

	HealthCheckInterval time.Duration

	MaxFailures    int
	CircuitTimeout time.Duration
	ResetTimeout   time.Duration
	SlowThreshold  time.Duration
	MaxSlowCount   int

	RequestTimeout  time.Duration
	ConnectTimeout  time.Duration
	ResponseTimeout time.Duration
}

func Load() (*Config, error) {
	loadEnvFile()
	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return config, nil
}

func loadConfig() (*Config, error) {
	port := getEnvRaw("PORT")
	maxFailures, err := getEnvIntRaw("MAX_FAILURES")
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_FAILURES: %w", err)
	}

	config := &Config{
		Port:            port,
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ApplicationAPIs: getApplicationAPIs(),
		BalancerType:    getEnv("BALANCER_TYPE", "round-robin"),

		HealthCheckInterval: getEnvDuration("HEALTH_CHECK_INTERVAL", "5s"),

		MaxFailures:    maxFailures,
		CircuitTimeout: getEnvDuration("CIRCUIT_TIMEOUT", "30s"),
		ResetTimeout:   getEnvDuration("RESET_TIMEOUT", "60s"),
		SlowThreshold:  getEnvDuration("SLOW_THRESHOLD", "5s"),
		MaxSlowCount:   getEnvInt("MAX_SLOW_COUNT", 3),

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
	var apis []string

	if apisEnv := os.Getenv("APPLICATION_APIS"); apisEnv != "" {
		parts := strings.Split(apisEnv, ",")
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				apis = append(apis, trimmed)
			}
		}
		return apis
	}

	for i := 1; i <= 10; i++ {
		if api := os.Getenv("API_" + strconv.Itoa(i)); api != "" {
			apis = append(apis, api)
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

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}
