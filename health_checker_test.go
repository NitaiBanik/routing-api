package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPHealthChecker_CheckClient(t *testing.T) {
	healthChecker := NewHTTPHealthChecker()

	tests := []struct {
		name           string
		serverResponse int
		serverError    bool
		expectedUp     bool
	}{
		{
			name:           "healthy server returns 200",
			serverResponse: http.StatusOK,
			expectedUp:     true,
		},
		{
			name:           "unhealthy server returns 500",
			serverResponse: http.StatusInternalServerError,
			expectedUp:     false,
		},
		{
			name:           "server returns 404",
			serverResponse: http.StatusNotFound,
			expectedUp:     false,
		},
		{
			name:        "server connection error",
			serverError: true,
			expectedUp:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if !tt.serverError {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.serverResponse)
				}))
				defer server.Close()
			}

			var baseURL string
			if tt.serverError {
				baseURL = "http://invalid-server:9999"
			} else {
				baseURL = server.URL
			}

			client := &defaultHTTPClient{
				Client: &http.Client{Timeout: 1 * time.Second},
				baseURL: baseURL,
				isUp:    true,
			}

			healthChecker.checkClient(client)
			assert.Equal(t, tt.expectedUp, client.IsUp())
		})
	}
}

func TestHTTPHealthChecker_CheckAllClients(t *testing.T) {
	healthChecker := NewHTTPHealthChecker()
	healthChanged := false
	onHealthChange := func() {
		healthChanged = true
	}

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server2.Close()

	client1 := &defaultHTTPClient{
		Client: &http.Client{Timeout: 1 * time.Second},
		baseURL: server1.URL,
		isUp:    false,
	}

	client2 := &defaultHTTPClient{
		Client: &http.Client{Timeout: 1 * time.Second},
		baseURL: server2.URL,
		isUp:    true,
	}

	clients := []HTTPClient{client1, client2}
	healthChecker.checkAllClients(clients, onHealthChange)

	assert.True(t, healthChanged)
	assert.True(t, client1.IsUp())
	assert.False(t, client2.IsUp())
}

func TestHTTPHealthChecker_Start(t *testing.T) {
	healthChecker := NewHTTPHealthChecker()
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &defaultHTTPClient{
		Client: &http.Client{Timeout: 1 * time.Second},
		baseURL: server.URL,
		isUp:    false,
	}

	clients := []HTTPClient{client}
	healthChanged := false
	onHealthChange := func() {
		healthChanged = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	healthChecker.Start(ctx, clients, 50*time.Millisecond, onHealthChange)
	time.Sleep(100 * time.Millisecond)

	assert.True(t, healthChanged)
	assert.True(t, client.IsUp())
}
