package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	handler := NewProxyHandler([]string{"http://localhost:8080"}, "round-robin")

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "health check works",
			method:         "GET",
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			handler.HealthHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "healthy", response.Status)
		})
	}
}

func TestProxyRequest(t *testing.T) {
	// Mock balancer that returns empty string (no servers configured)
	mockBalancer := &MockBalancer{serverURL: ""}
	handler := NewProxyHandlerWithDeps(mockBalancer, &defaultHTTPClient{Client: &http.Client{}})

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "no apis configured returns 500",
			method:         "POST",
			body:           `{"message": "Hello"}`,
			expectedStatus: 500,
		},
		{
			name:           "get request with no apis returns 500",
			method:         "GET",
			body:           "",
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/testapi", bytes.NewBuffer([]byte(tt.body)))
			w := httptest.NewRecorder()

			handler.ProxyRequest(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

type MockBalancer struct {
	serverURL string
}

func (m *MockBalancer) Next() string {
	return m.serverURL
}

func TestRoundRobinDistribution(t *testing.T) {
	// Create multiple mock servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"server": "1"})
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"server": "2"})
	}))
	defer server2.Close()

	balancer := &roundRobinLoadBalancer{servers: []string{server1.URL, server2.URL}}
	handler := NewProxyHandlerWithDeps(balancer, &defaultHTTPClient{Client: &http.Client{}})

	tests := []struct {
		name          string
		requestCount  int
		expectedOrder []string
	}{
		{
			name:          "round robin works",
			requestCount:  4,
			expectedOrder: []string{"1", "2", "1", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := make([]string, tt.requestCount)
			for i := 0; i < tt.requestCount; i++ {
				req, _ := http.NewRequest("POST", "/testapi", bytes.NewBuffer([]byte(`{"test": true}`)))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler.ProxyRequest(w, req)

				assert.Equal(t, http.StatusOK, w.Code)

				var response map[string]string
				json.Unmarshal(w.Body.Bytes(), &response)
				responses[i] = response["server"]
			}

			for i, expected := range tt.expectedOrder {
				assert.Equal(t, expected, responses[i])
			}
		})
	}
}

func TestLoadBalancerFactory(t *testing.T) {
	factory := NewLoadBalancerFactory()
	servers := []string{"server1", "server2"}

	tests := []struct {
		name         string
		balancerType string
	}{
		{
			name:         "round-robin balancer",
			balancerType: "round-robin",
		},
		{
			name:         "unknown balancer defaults to round-robin",
			balancerType: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balancer := factory.CreateLoadBalancer(tt.balancerType, servers)
			assert.NotNil(t, balancer)

			// Test that it returns servers in round-robin fashion
			first := balancer.Next()
			second := balancer.Next()
			third := balancer.Next()

			assert.Equal(t, "server1", first)
			assert.Equal(t, "server2", second)
			assert.Equal(t, "server1", third) // should wrap around
		})
	}
}
