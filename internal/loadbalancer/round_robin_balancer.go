package loadbalancer

import (
	"context"
	"net/http"
	"sync"
	"time"

	"routing-api/internal/circuit"
	"routing-api/internal/health"
	"routing-api/internal/logger"
)

type roundRobinLoadBalancer struct {
	clients          []health.HTTPClient
	availableClients []health.HTTPClient
	currentIndex     int
	mutex            sync.RWMutex
	logger           logger.Logger
}

func (r *roundRobinLoadBalancer) Next() health.HTTPClient {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.availableClients) == 0 {
		return nil
	}

	client := r.availableClients[r.currentIndex]
	r.currentIndex = (r.currentIndex + 1) % len(r.availableClients)
	return client
}

func newRoundRobinLoadBalancer(servers []string, circuitConfig circuit.CircuitBreakerConfig, logger logger.Logger) *roundRobinLoadBalancer {
	clients := make([]health.HTTPClient, len(servers))
	availableClients := make([]health.HTTPClient, len(servers))

	for i, serverURL := range servers {
		baseClient := &health.DefaultHTTPClient{
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
			BaseURL: serverURL,
			Up:      true,
		}
		circuitBreakerClient := circuit.NewCircuitBreakerClient(baseClient, circuitConfig)
		clients[i] = circuitBreakerClient
		availableClients[i] = circuitBreakerClient
	}

	return &roundRobinLoadBalancer{
		clients:          clients,
		availableClients: availableClients,
		logger:           logger,
	}
}

func (r *roundRobinLoadBalancer) StartHealthChecks(ctx context.Context, interval time.Duration) {
	healthChecker := health.NewHTTPHealthChecker(r.logger)
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
