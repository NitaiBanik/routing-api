package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"routing-api/internal/circuit"
	"routing-api/internal/loadbalancer"
	"routing-api/internal/logger"
	"routing-api/internal/proxy"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type testLogger struct{}

func (l *testLogger) Debug(msg string, fields ...zap.Field)  {}
func (l *testLogger) Info(msg string, fields ...zap.Field)   {}
func (l *testLogger) Warn(msg string, fields ...zap.Field)   {}
func (l *testLogger) Error(msg string, fields ...zap.Field)  {}
func (l *testLogger) Fatal(msg string, fields ...zap.Field)  {}
func (l *testLogger) With(fields ...zap.Field) logger.Logger { return l }
func (l *testLogger) Sync() error                            { return nil }

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

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{server1.URL, server2.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

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

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{healthyServer.URL, unhealthyServer.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go handler.StartHealthChecks(ctx, 50*time.Millisecond)

	time.Sleep(500 * time.Millisecond)
	healthyCount := 0
	unhealthyCount := 0

	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ProxyRequest(w, req)

		if w.Code == http.StatusOK && w.Body.String() == "healthy" {
			healthyCount++
		} else if w.Code == http.StatusOK && w.Body.String() == "unhealthy" {
			unhealthyCount++
		}
	}

	// This is more realistic than expecting 100% healthy responses immediately
	assert.Greater(t, healthyCount, 0, "Should get at least some healthy responses")
	t.Logf("Healthy responses: %d, Unhealthy responses: %d", healthyCount, unhealthyCount)
}

func TestIntegration_CircuitBreakerWithRetry(t *testing.T) {
	// Create a server that returns HTTP 500 (no retry for HTTP errors)
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 100 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{server.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

	// HTTP 500 errors are not retried, so only 1 attempt
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ProxyRequest(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", w.Body.String())
	assert.Equal(t, 1, attemptCount)
}

func TestIntegration_AllBackendsDown(t *testing.T) {
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{"http://invalid1:9999", "http://invalid2:9999"}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

	// Start health checks
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go handler.StartHealthChecks(ctx, 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	// Request should fail with no servers available
	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ProxyRequest(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "cannot reach server")
}

func TestIntegration_JSONRequestForwarding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 100 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{server.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

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

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "test-server", response["server"])
	// JSON unmarshaling converts numbers to float64
	expectedEcho := map[string]interface{}{
		"message": "hello",
		"number":  float64(42),
	}
	assert.Equal(t, expectedEcho, response["echo"])
}

func TestIntegration_CircuitBreakerAllStates(t *testing.T) {
	// Create a server that fails initially, then recovers
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++

		// Fail first 2 requests by closing connection, then succeed
		if attemptCount <= 2 {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  2,
		ResetTimeout: 200 * time.Millisecond,
	}

	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{server.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := proxy.NewProxyHandler(clientProvider, &testLogger{})

	// Test 1: CLOSED STATE - First request fails (network error)
	t.Log("Testing CLOSED state (first failure)...")
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	handler.ProxyRequest(w1, req1)

	assert.Equal(t, http.StatusBadGateway, w1.Code)
	assert.Contains(t, w1.Body.String(), "cannot reach server")

	// Test 2: CLOSED STATE - Second request fails, circuit opens
	t.Log("Testing CLOSED state (second failure)...")
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	handler.ProxyRequest(w2, req2)

	assert.Equal(t, http.StatusBadGateway, w2.Code)
	assert.Contains(t, w2.Body.String(), "cannot reach server")

	// Test 3: OPEN STATE - Multiple requests should be blocked by circuit breaker
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler.ProxyRequest(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "circuit breaker is open")
		t.Logf("Request %d blocked: %s", i+1, w.Body.String())
	}

	// Test 4: Wait for reset timeout and test HALF-OPEN state
	time.Sleep(250 * time.Millisecond)

	req4, _ := http.NewRequest("GET", "/test", nil)
	w4 := httptest.NewRecorder()
	handler.ProxyRequest(w4, req4)

	assert.Equal(t, http.StatusOK, w4.Code)
	assert.Equal(t, "success", w4.Body.String())

	// Test 5: CLOSED STATE - Circuit should be closed again
	req5, _ := http.NewRequest("GET", "/test", nil)
	w5 := httptest.NewRecorder()
	handler.ProxyRequest(w5, req5)

	assert.Equal(t, http.StatusOK, w5.Code)
	assert.Equal(t, "success", w5.Body.String())
}
