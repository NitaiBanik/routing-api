package circuit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"routing-api/internal/health"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreakerClient_NetworkFailure(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 100 * time.Millisecond},
		BaseURL: "http://localhost:9999",
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := circuitBreakerClient.Do(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestCircuitBreakerClient_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: server.URL,
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := circuitBreakerClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestCircuitBreakerClient_CircuitBreakerOpen(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://invalid-server:9999",
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	for i := 0; i < 2; i++ {
		resp, err := circuitBreakerClient.Do(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	}

	resp, err := circuitBreakerClient.Do(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &CircuitBreakerError{}, err)
}

func TestCircuitBreakerClient_CircuitBreakerRecovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: server.URL,
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 50 * time.Millisecond,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := circuitBreakerClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.True(t, circuitBreakerClient.IsUp())
}

func TestCircuitBreakerClient_IsUp(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 60 * time.Second,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	assert.True(t, circuitBreakerClient.IsUp())
}

func TestCircuitBreakerClient_SetUp(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      true,
	}

	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 60 * time.Second,
	}

	circuitBreakerClient := NewCircuitBreakerClient(baseClient, circuitConfig)

	assert.NotPanics(t, func() {
		circuitBreakerClient.SetUp(true)
		circuitBreakerClient.SetUp(false)
	})
}
