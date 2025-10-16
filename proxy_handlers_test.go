package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	mockBalancer := &MockLoadBalancer{client: nil}
	handler := NewProxyHandlerWithDeps(mockBalancer)

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
			req, _ := http.NewRequest(tt.method, "/any-endpoint", bytes.NewBuffer([]byte(tt.body)))
			w := httptest.NewRecorder()

			handler.ProxyRequest(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

type MockLoadBalancer struct {
	client HTTPClient
}

func (m *MockLoadBalancer) Next() HTTPClient {
	return m.client
}

type MockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestRoundRobinDistribution(t *testing.T) {
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

	balancer := newRoundRobinLoadBalancer([]string{server1.URL, server2.URL})
	handler := NewProxyHandlerWithDeps(balancer)

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
				req, _ := http.NewRequest("POST", "/echo", bytes.NewBuffer([]byte(`{"test": true}`)))
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

			// Test that it returns clients in round-robin fashion
			first := balancer.Next()
			second := balancer.Next()
			third := balancer.Next()

			assert.NotNil(t, first)
			assert.NotNil(t, second)
			assert.NotNil(t, third)

			// Test that we get different clients (they should have different base URLs)
			assert.NotEqual(t, first, second)
			assert.Equal(t, first, third) // should wrap around
		})
	}
}

func TestHTTPClientWithBaseURL(t *testing.T) {
	client := &defaultHTTPClient{
		Client:  &http.Client{},
		baseURL: "http://example.com",
	}

	req, _ := http.NewRequest("GET", "/test", nil)
	originalURL := req.URL.String()

	client.Client = &http.Client{
		Transport: &mockTransport{},
	}

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.NotEqual(t, originalURL, req.URL.String())
	assert.Contains(t, req.URL.String(), "http://example.com")
}

type mockTransport struct{}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
		Header:     make(http.Header),
	}, nil
}
