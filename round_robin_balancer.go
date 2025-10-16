package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type roundRobinLoadBalancer struct {
	clients          []HTTPClient
	availableClients []HTTPClient
	currentIndex     int
	mutex            sync.RWMutex
}

func (r *roundRobinLoadBalancer) Next() HTTPClient {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.availableClients) == 0 {
		return nil
	}

	client := r.availableClients[r.currentIndex%len(r.availableClients)]
	r.currentIndex = (r.currentIndex + 1) % len(r.availableClients)
	return client
}

func newRoundRobinLoadBalancer(servers []string, retryConfig RetryConfig, circuitConfig CircuitBreakerConfig) *roundRobinLoadBalancer {
	clients := make([]HTTPClient, len(servers))
	availableClients := make([]HTTPClient, len(servers))

	for i, serverURL := range servers {
		baseClient := &defaultHTTPClient{
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
			baseURL: serverURL,
			isUp:    true,
		}
		retryableClient := NewRetryableClient(baseClient, retryConfig, circuitConfig)
		clients[i] = retryableClient
		availableClients[i] = retryableClient
	}

	return &roundRobinLoadBalancer{
		clients:          clients,
		availableClients: availableClients,
	}
}

func (r *roundRobinLoadBalancer) StartHealthChecks(ctx context.Context, interval time.Duration) {
	healthChecker := NewHTTPHealthChecker()
	go healthChecker.Start(ctx, r.clients, interval, r.updateAvailableClients)
}

func (r *roundRobinLoadBalancer) updateAvailableClients() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	available := make([]HTTPClient, 0)
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
