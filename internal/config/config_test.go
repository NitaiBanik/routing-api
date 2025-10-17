package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "3000", cfg.Port)
	assert.NotEmpty(t, cfg.ApplicationAPIs)
}

func TestConfigValidation(t *testing.T) {
	os.Clearenv()

	_, err := Load()
	assert.NoError(t, err)
}
