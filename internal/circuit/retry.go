package circuit

import (
	"net/http"
	"time"

	"routing-api/internal/health"
)

type RetryConfig struct {
	MaxAttempts int
	Delay       time.Duration
}

type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
}

type RetryableClient struct {
	client         health.HTTPClient
	config         RetryConfig
	circuitBreaker *CircuitBreaker
}

func NewRetryableClient(client health.HTTPClient, retryConfig RetryConfig, circuitConfig CircuitBreakerConfig) *RetryableClient {
	circuitBreaker := NewCircuitBreaker(circuitConfig.MaxFailures, circuitConfig.ResetTimeout)

	return &RetryableClient{
		client:         client,
		config:         retryConfig,
		circuitBreaker: circuitBreaker,
	}
}

func NewRetryableClientWithCircuitBreaker(client health.HTTPClient, retryConfig RetryConfig, circuitBreaker *CircuitBreaker) *RetryableClient {
	return &RetryableClient{
		client:         client,
		config:         retryConfig,
		circuitBreaker: circuitBreaker,
	}
}

func (rc *RetryableClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < rc.config.MaxAttempts; attempt++ {
		var resp *http.Response
		err := rc.circuitBreaker.Execute(func() error {
			var execErr error
			resp, execErr = rc.client.Do(req)
			return execErr
		})

		if err != nil {
			lastErr = err

			if _, isCircuitBreakerErr := err.(*CircuitBreakerError); isCircuitBreakerErr {
				return nil, err
			}

			if attempt == rc.config.MaxAttempts-1 {
				return nil, err
			}

			time.Sleep(rc.config.Delay)
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

func (rc *RetryableClient) IsUp() bool {
	return !rc.circuitBreaker.IsOpen()
}

func (rc *RetryableClient) SetUp(isUp bool) {
}

func (rc *RetryableClient) GetBaseURL() string {
	return rc.client.GetBaseURL()
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		Delay:       100 * time.Millisecond,
	}
}
