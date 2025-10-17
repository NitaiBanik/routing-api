package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"routing-api/internal/circuit"

	"github.com/stretchr/testify/assert"
)

func TestRoundRobinLoadBalancer_UpdateAvailableClients(t *testing.T) {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := newRoundRobinLoadBalancer([]string{server1.URL, server2.URL}, retryConfig, circuitConfig)
	assert.Equal(t, 2, len(balancer.availableClients))
}

func TestRoundRobinLoadBalancer_AllClientsDown(t *testing.T) {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	balancer := newRoundRobinLoadBalancer([]string{"http://localhost:8080", "http://localhost:8081"}, retryConfig, circuitConfig)
	assert.Equal(t, 2, len(balancer.availableClients))

	client := balancer.Next()
	assert.NotNil(t, client)
}

func TestRoundRobinLoadBalancer_IndexManagement(t *testing.T) {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := newRoundRobinLoadBalancer([]string{server1.URL, server2.URL}, retryConfig, circuitConfig)

	// Test round robin distribution
	client1 := balancer.Next()
	client2 := balancer.Next()
	client3 := balancer.Next()

	assert.NotNil(t, client1)
	assert.NotNil(t, client2)
	assert.NotNil(t, client3)

	// Should cycle through clients
	assert.Equal(t, client1, client3)
	assert.NotEqual(t, client1, client2)
}

func TestRoundRobinLoadBalancer_IndexAdjustmentOnClientRemoval(t *testing.T) {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := newRoundRobinLoadBalancer([]string{server1.URL, server2.URL}, retryConfig, circuitConfig)
	balancer.currentIndex = 1
	balancer.updateAvailableClients()

	assert.True(t, balancer.currentIndex < len(balancer.availableClients))
}

func TestRoundRobinLoadBalancer_ConcurrentAccess(t *testing.T) {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	balancer := newRoundRobinLoadBalancer([]string{server.URL}, retryConfig, circuitConfig)

	// Test concurrent access to Next() and updateAvailableClients()
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			balancer.Next()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			balancer.updateAvailableClients()
		}
		done <- true
	}()

	<-done
	<-done

	// Should not panic and should have valid state
	client := balancer.Next()
	assert.NotNil(t, client)
}
