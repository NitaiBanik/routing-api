package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIntegration_LoadBalancingWithFailures(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("server1"))
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("server2"))
	}))
	defer server2.Close()

	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	balancer := newRoundRobinLoadBalancer([]string{server1.URL, server2.URL}, retryConfig, circuitConfig)
	clientProvider := NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider)

	responses := make(map[string]int)
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ProxyRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		responses[w.Body.String()]++
	}

	assert.True(t, responses["server1"] > 0)
	assert.True(t, responses["server2"] > 0)
}

func TestIntegration_HealthCheckIntegration(t *testing.T) {
	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
		}
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("unhealthy"))
		}
	}))
	defer unhealthyServer.Close()

	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	balancer := newRoundRobinLoadBalancer([]string{healthyServer.URL, unhealthyServer.URL}, retryConfig, circuitConfig)
	clientProvider := NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider)

	// Start health checks
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go handler.StartHealthChecks(ctx, 50*time.Millisecond)

	// Wait for health checks to run
	time.Sleep(100 * time.Millisecond)

	// All requests should go to healthy server
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ProxyRequest(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "healthy", w.Body.String())
	}
}

func TestIntegration_CircuitBreakerWithRetry(t *testing.T) {
	// Create a server that fails initially then succeeds
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		}
	}))
	defer server.Close()

	retryConfig := RetryConfig{
		MaxAttempts: 3,
		Delay:       10 * time.Millisecond,
	}
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 100 * time.Millisecond,
	}

	balancer := newRoundRobinLoadBalancer([]string{server.URL}, retryConfig, circuitConfig)
	clientProvider := NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider)

	// First request should succeed after retries
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ProxyRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", w.Body.String())
	assert.Equal(t, 3, attemptCount)
}

func TestIntegration_AllBackendsDown(t *testing.T) {
	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	balancer := newRoundRobinLoadBalancer([]string{"http://invalid1:9999", "http://invalid2:9999"}, retryConfig, circuitConfig)
	clientProvider := NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider)

	// Start health checks
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go handler.StartHealthChecks(ctx, 50*time.Millisecond)

	// Wait for health checks to mark servers as down
	time.Sleep(100 * time.Millisecond)

	// Request should fail with no servers available
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ProxyRequest(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "no servers configured")
}

func TestIntegration_JSONRequestForwarding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse and echo back the request body
		var requestData map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestData)

		response := map[string]interface{}{
			"echo":   requestData,
			"server": "test-server",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	retryConfig := DefaultRetryConfig()
	circuitConfig := CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	balancer := newRoundRobinLoadBalancer([]string{server.URL}, retryConfig, circuitConfig)
	clientProvider := NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider)

	// Send JSON request
	requestData := map[string]interface{}{
		"message": "hello",
		"number":  42,
	}

	requestBody, _ := json.Marshal(requestData)
	req, _ := http.NewRequest("POST", "/api/test", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ProxyRequest(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "test-server", response["server"])
	assert.Equal(t, requestData, response["echo"])
}
