package circuit

import (
	"net/http"
	"time"

	"routing-api/internal/health"
)

type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
}

type CircuitBreakerClient struct {
	client         health.HTTPClient
	circuitBreaker *CircuitBreaker
}

func NewCircuitBreakerClient(client health.HTTPClient, circuitConfig CircuitBreakerConfig) *CircuitBreakerClient {
	circuitBreaker := NewCircuitBreaker(circuitConfig.MaxFailures, circuitConfig.ResetTimeout)

	return &CircuitBreakerClient{
		client:         client,
		circuitBreaker: circuitBreaker,
	}
}

func (cbc *CircuitBreakerClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := cbc.circuitBreaker.Execute(func() error {
		var execErr error
		resp, execErr = cbc.client.Do(req)
		return execErr
	})

	return resp, err
}

func (cbc *CircuitBreakerClient) IsUp() bool {
	return cbc.client.IsUp()
}

func (cbc *CircuitBreakerClient) SetUp(isUp bool) {
	cbc.client.SetUp(isUp)

	if isUp {
		cbc.circuitBreaker.mutex.Lock()
		cbc.circuitBreaker.failureCount = 0
		cbc.circuitBreaker.slowCount = 0
		cbc.circuitBreaker.state = StateClosed
		cbc.circuitBreaker.mutex.Unlock()
	}
}

func (cbc *CircuitBreakerClient) GetBaseURL() string {
	return cbc.client.GetBaseURL()
}
