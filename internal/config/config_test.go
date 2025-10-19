package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	os.Clearenv()

	// Set required environment variables
	os.Setenv("PORT", "3000")
	os.Setenv("MAX_FAILURES", "5")
	os.Setenv("API_1", "http://localhost:8080")
	os.Setenv("API_2", "http://localhost:8081")
	os.Setenv("API_3", "http://localhost:8082")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "3000", cfg.Port)
	assert.NotEmpty(t, cfg.ApplicationAPIs)
}

func TestConfigValidation(t *testing.T) {
	os.Clearenv()

	// Set required environment variables
	os.Setenv("PORT", "3000")
	os.Setenv("MAX_FAILURES", "5")
	os.Setenv("API_1", "http://localhost:8080")
	os.Setenv("API_2", "http://localhost:8081")
	os.Setenv("API_3", "http://localhost:8082")

	_, err := Load()
	assert.NoError(t, err)
}

func TestConfigLoadWithDefaults(t *testing.T) {
	os.Clearenv()

	os.Setenv("PORT", "3000")
	os.Setenv("MAX_FAILURES", "5")
	os.Setenv("API_1", "http://localhost:8080")
	os.Setenv("API_2", "http://localhost:8081")
	os.Setenv("API_3", "http://localhost:8082")

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "3000", cfg.Port)
	assert.Equal(t, 5, cfg.MaxFailures)
	assert.NotEmpty(t, cfg.ApplicationAPIs)
}

func TestConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing port",
			setupEnv: func() {
				os.Clearenv()
				os.Setenv("MAX_FAILURES", "5")
				os.Setenv("API_1", "http://localhost:8080")
			},
			expectError: true,
			errorMsg:    "port cannot be empty",
		},
		{
			name: "no APIs configured",
			setupEnv: func() {
				os.Clearenv()
				os.Setenv("PORT", "3000")
				os.Setenv("MAX_FAILURES", "5")
			},
			expectError: true,
			errorMsg:    "at least one application API must be configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			_, err := Load()
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
