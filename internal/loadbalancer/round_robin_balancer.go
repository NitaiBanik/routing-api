package loadbalancer

import (
	"context"
	"sync"
	"time"

	"routing-api/internal/circuit"
	"routing-api/internal/health"
)

type roundRobinLoadBalancer struct {
	clients          []health.HTTPClient
	availableClients []health.HTTPClient
	currentIndex     int
	mutex            sync.RWMutex
}

func (r *roundRobinLoadBalancer) Next() health.HTTPClient {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.availableClients) == 0 {
		return nil
	}

	client := r.availableClients[r.currentIndex%len(r.availableClients)]
	r.currentIndex = (r.currentIndex + 1) % len(r.availableClients)
	return client
}

func newRoundRobinLoadBalancer(config LoadBalancerConfig) *roundRobinLoadBalancer {
	clients := make([]health.HTTPClient, len(config.Servers))
	availableClients := make([]health.HTTPClient, len(config.Servers))

	for i, serverURL := range config.Servers {
		baseClient := health.NewDefaultHTTPClient(serverURL, config.RequestTimeout, config.ConnectTimeout)
		circuitBreaker := circuit.NewCircuitBreakerWithSlowThreshold(
			config.CircuitConfig.MaxFailures,
			config.CircuitConfig.ResetTimeout,
			config.SlowThreshold,
			config.MaxSlowCount,
		)
		retryableClient := circuit.NewRetryableClientWithCircuitBreaker(baseClient, config.RetryConfig, circuitBreaker)
		clients[i] = retryableClient
		availableClients[i] = retryableClient
	}

	return &roundRobinLoadBalancer{
		clients:          clients,
		availableClients: availableClients,
	}
}

func (r *roundRobinLoadBalancer) StartHealthChecks(ctx context.Context, interval time.Duration) {
	healthChecker := health.NewHTTPHealthChecker()
	go healthChecker.Start(ctx, r.clients, interval, r.updateAvailableClients)
}

func (r *roundRobinLoadBalancer) updateAvailableClients() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	available := make([]health.HTTPClient, 0)
	for _, client := range r.clients {
		if client.IsUp() {
			available = append(available, client)
		}
	}
	r.availableClients = available

	if len(r.availableClients) > 0 {
		r.currentIndex = r.currentIndex % len(r.availableClients)
	} else {
		r.currentIndex = 0
	}
}
