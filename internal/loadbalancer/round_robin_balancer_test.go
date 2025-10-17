package loadbalancer

import (
	"net/http"
	"net/http/httptest"
	"routing-api/internal/circuit"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createTestLoadBalancer(servers []string) *roundRobinLoadBalancer {
	retryConfig := circuit.DefaultRetryConfig()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}
	config := LoadBalancerConfig{
		BalancerType:   "round-robin",
		Servers:        servers,
		RetryConfig:    retryConfig,
		CircuitConfig:  circuitConfig,
		RequestTimeout: 30 * time.Second,
		ConnectTimeout: 5 * time.Second,
		SlowThreshold:  10 * time.Second,
		MaxSlowCount:   10,
	}
	return newRoundRobinLoadBalancer(config)
}

func TestRoundRobinLoadBalancer_UpdateAvailableClients(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := createTestLoadBalancer([]string{server1.URL, server2.URL})
	assert.Equal(t, 2, len(balancer.availableClients))
}

func TestRoundRobinLoadBalancer_AllClientsDown(t *testing.T) {
	balancer := createTestLoadBalancer([]string{"http://localhost:8080", "http://localhost:8081"})
	assert.Equal(t, 2, len(balancer.availableClients))

	client := balancer.Next()
	assert.NotNil(t, client)
}

func TestRoundRobinLoadBalancer_IndexManagement(t *testing.T) {

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := createTestLoadBalancer([]string{server1.URL, server2.URL})

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

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	balancer := createTestLoadBalancer([]string{server1.URL, server2.URL})
	balancer.currentIndex = 1
	balancer.updateAvailableClients()

	assert.True(t, balancer.currentIndex < len(balancer.availableClients))
}

func TestRoundRobinLoadBalancer_ConcurrentAccess(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	balancer := createTestLoadBalancer([]string{server.URL})

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
