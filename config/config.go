package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string
	Environment     string
	LogLevel        string
	ApplicationAPIs []string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "3000"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ApplicationAPIs: getApplicationAPIs(),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
