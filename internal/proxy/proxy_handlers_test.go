package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"routing-api/internal/circuit"
	"routing-api/internal/health"
	"routing-api/internal/loadbalancer"
	"routing-api/internal/logger"

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

func TestHealthHandler(t *testing.T) {
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}
	factory := loadbalancer.NewLoadBalancerFactory()
	loadBalancer := factory.CreateLoadBalancer("round-robin", []string{"http://localhost:8080"}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(loadBalancer)
	handler := NewProxyHandler(clientProvider, &testLogger{})

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
	mockProvider := &MockClientProvider{client: nil}
	handler := NewProxyHandler(mockProvider, &testLogger{})

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

type MockClientProvider struct {
	client health.HTTPClient
}

func (m *MockClientProvider) GetClient() health.HTTPClient {
	return m.client
}

func (m *MockClientProvider) StartHealthChecks(ctx context.Context, interval time.Duration) {
	// Mock implementation - do nothing
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
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"server": "1"})
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"server": "2"})
	}))
	defer server2.Close()

	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}
	factory := loadbalancer.NewLoadBalancerFactory()
	balancer := factory.CreateLoadBalancer("round-robin", []string{server1.URL, server2.URL}, circuitConfig, &testLogger{})
	clientProvider := loadbalancer.NewLoadBalancerAdapter(balancer)
	handler := NewProxyHandler(clientProvider, &testLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go handler.StartHealthChecks(ctx, 100*time.Millisecond)
	time.Sleep(200 * time.Millisecond)

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
	factory := loadbalancer.NewLoadBalancerFactory()
	circuitConfig := circuit.CircuitBreakerConfig{
		MaxFailures:  5,
		ResetTimeout: 60 * time.Second,
	}

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	servers := []string{server1.URL, server2.URL}

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
			balancer := factory.CreateLoadBalancer(tt.balancerType, servers, circuitConfig, &testLogger{})
			assert.NotNil(t, balancer)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go balancer.StartHealthChecks(ctx, 100*time.Millisecond)
			time.Sleep(200 * time.Millisecond)
			first := balancer.Next()
			second := balancer.Next()
			third := balancer.Next()

			assert.NotNil(t, first)
			assert.NotNil(t, second)
			assert.NotNil(t, third)
		})
	}
}

func TestHTTPClientWithBaseURL(t *testing.T) {
	client := &health.DefaultHTTPClient{
		Client:  &http.Client{},
		BaseURL: "http://example.com",
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
