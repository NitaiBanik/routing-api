package circuit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"routing-api/internal/health"

	"github.com/stretchr/testify/assert"
)

func TestRetryableClient_RetryOnNetworkFailure(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 100 * time.Millisecond},
		BaseURL: "http://localhost:9999",
		Up:      true,
	}

	retryConfig := RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := retryableClient.Do(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRetryableClient_HTTPErrorNoRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: server.URL,
		Up:      true,
	}

	retryConfig := RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := retryableClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRetryableClient_CircuitBreakerOpen(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://invalid-server:9999",
		Up:      true,
	}

	retryConfig := RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	for i := 0; i < 2; i++ {
		resp, err := retryableClient.Do(req)
		assert.Error(t, err)
		assert.Nil(t, resp)
	}

	resp, err := retryableClient.Do(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.IsType(t, &CircuitBreakerError{}, err)
}

func TestRetryableClient_CircuitBreakerRecovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: server.URL,
		Up:      true,
	}

	retryConfig := RetryConfig{
		MaxAttempts: 1,
		Delay:       10 * time.Millisecond,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  1,
		ResetTimeout: 50 * time.Millisecond,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := retryableClient.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.True(t, retryableClient.IsUp())
}

func TestRetryableClient_IsUp(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      true,
	}

	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 60 * time.Second,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	assert.True(t, retryableClient.IsUp())
}

func TestRetryableClient_SetUp(t *testing.T) {
	baseClient := &health.DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      true,
	}

	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 60 * time.Second,
	}

	retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)

	assert.NotPanics(t, func() {
		retryableClient.SetUp(true)
		retryableClient.SetUp(false)
	})
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.Delay)
}
