package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	handler := NewHandler([]string{"http://localhost:8080"})

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
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, "healthy", response.Status)
			assert.Equal(t, "routing-api", response.Service)
			assert.NotNil(t, response.Timestamp)
		})
	}
}

func TestProxyRequest(t *testing.T) {
	handler := NewHandler([]string{})

	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "no apis configured returns 500",
			method:         "POST",
			body:           map[string]interface{}{"message": "Hello"},
			expectedStatus: 500,
		},
		{
			name:           "get request with no apis returns 500",
			method:         "GET",
			body:           nil,
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.body != nil {
				body, _ = json.Marshal(tt.body)
			}

			req, _ := http.NewRequest(tt.method, "/testapi", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			handler.ProxyRequest(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
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

	handler := NewHandler([]string{server1.URL, server2.URL})

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
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				responses[i] = response["server"]
			}

			for i, expected := range tt.expectedOrder {
				assert.Equal(t, expected, responses[i])
			}
		})
	}
}
