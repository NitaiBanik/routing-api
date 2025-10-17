package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultHTTPClient_Do(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := &DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: server.URL,
		Up:      true,
	}

	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDefaultHTTPClient_IsUp(t *testing.T) {
	client := &DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      true,
	}

	assert.True(t, client.IsUp())

	client.Up = false
	assert.False(t, client.IsUp())
}

func TestDefaultHTTPClient_SetUp(t *testing.T) {
	client := &DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      false,
	}

	client.SetUp(true)
	assert.True(t, client.IsUp())

	client.SetUp(false)
	assert.False(t, client.IsUp())
}

func TestDefaultHTTPClient_ConcurrentAccess(t *testing.T) {
	client := &DefaultHTTPClient{
		Client:  &http.Client{Timeout: 5 * time.Second},
		BaseURL: "http://example.com",
		Up:      false,
	}

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			client.SetUp(i%2 == 0)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			client.IsUp()
		}
		done <- true
	}()

	<-done
	<-done
	_ = client.IsUp()
}
