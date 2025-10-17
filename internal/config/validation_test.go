package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "valid configuration",
			envVars: map[string]string{
				"PORT":                  "8080",
				"APPLICATION_APIS":      "http://localhost:8081,http://localhost:8082",
				"BALANCER_TYPE":         "round-robin",
				"HEALTH_CHECK_INTERVAL": "30s",
				"MAX_RETRIES":           "3",
				"RETRY_DELAY":           "100ms",
				"MAX_FAILURES":          "5",
				"RESET_TIMEOUT":         "60s",
			},
			expectError: false,
		},
		{
			name: "missing required PORT",
			envVars: map[string]string{
				"APPLICATION_APIS": "http://localhost:8081",
			},
			expectError: true,
		},
		{
			name: "missing required APPLICATION_APIS",
			envVars: map[string]string{
				"PORT": "8080",
			},
			expectError: true,
		},
		{
			name: "invalid PORT format",
			envVars: map[string]string{
				"PORT":             "invalid-port",
				"APPLICATION_APIS": "http://localhost:8081",
			},
			expectError: true,
		},
		{
			name: "invalid MAX_RETRIES",
			envVars: map[string]string{
				"PORT":             "8080",
				"APPLICATION_APIS": "http://localhost:8081",
				"MAX_RETRIES":      "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid MAX_FAILURES",
			envVars: map[string]string{
				"PORT":             "8080",
				"APPLICATION_APIS": "http://localhost:8081",
				"MAX_FAILURES":     "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := LoadWithDefaults(false)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("PORT", "8080")
	os.Setenv("APPLICATION_APIS", "http://localhost:8081")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "round-robin", cfg.BalancerType)
	assert.Equal(t, "5s", cfg.HealthCheckInterval.String())
	assert.Equal(t, 5, cfg.MaxFailures)
	assert.Equal(t, "1m0s", cfg.ResetTimeout.String())
}

func TestGetEnvInt_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{
			name:         "valid integer",
			envValue:     "42",
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "invalid integer",
			envValue:     "not-a-number",
			defaultValue: 10,
			expected:     10,
		},
		{
			name:         "empty value",
			envValue:     "",
			defaultValue: 5,
			expected:     5,
		},
		{
			name:         "negative integer",
			envValue:     "-5",
			defaultValue: 0,
			expected:     -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv("TEST_INT", tt.envValue)
			}

			result := getEnvInt("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvFloat_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue float64
		expected     float64
	}{
		{
			name:         "valid float",
			envValue:     "3.14",
			defaultValue: 0.0,
			expected:     3.14,
		},
		{
			name:         "invalid float",
			envValue:     "not-a-float",
			defaultValue: 1.0,
			expected:     1.0,
		},
		{
			name:         "empty value",
			envValue:     "",
			defaultValue: 2.5,
			expected:     2.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv("TEST_FLOAT", tt.envValue)
			}

			result := getEnvFloat("TEST_FLOAT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expectError  bool
	}{
		{
			name:         "valid duration",
			envValue:     "30s",
			defaultValue: "10s",
			expectError:  false,
		},
		{
			name:         "invalid duration",
			envValue:     "invalid-duration",
			defaultValue: "5s",
			expectError:  false, // Should fall back to default
		},
		{
			name:         "empty value",
			envValue:     "",
			defaultValue: "1m",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			if tt.envValue != "" {
				os.Setenv("TEST_DURATION", tt.envValue)
			}

			result := getEnvDuration("TEST_DURATION", tt.defaultValue)
			assert.NotNil(t, result)
		})
	}
}
