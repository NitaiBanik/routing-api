package health

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type HealthChecker interface {
	Start(ctx context.Context, clients []HTTPClient, interval time.Duration, onHealthChange func())
}

type httpHealthChecker struct {
	checkPath string
}

func NewHTTPHealthChecker() *httpHealthChecker {
	return &httpHealthChecker{
		checkPath: "/health",
	}
}

func (h *httpHealthChecker) Start(ctx context.Context, clients []HTTPClient, interval time.Duration, onHealthChange func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.checkAllClients(clients, onHealthChange)
		}
	}
}

func (h *httpHealthChecker) checkAllClients(clients []HTTPClient, onHealthChange func()) {
	var wg sync.WaitGroup
	healthChanged := false

	for _, client := range clients {
		wg.Add(1)
		go func(c HTTPClient) {
			defer wg.Done()
			oldStatus := c.IsUp()
			h.checkClient(c)
			newStatus := c.IsUp()
			if oldStatus != newStatus {
				healthChanged = true
			}
		}(client)
	}

	wg.Wait()

	if healthChanged {
		onHealthChange()
	}
}

func (h *httpHealthChecker) checkClient(client HTTPClient) {
	if defaultClient, ok := client.(*DefaultHTTPClient); ok {
		req, err := http.NewRequest("GET", defaultClient.BaseURL+h.checkPath, nil)
		if err != nil {
			client.SetUp(false)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := defaultClient.Client.Do(req)
		if err != nil {
			client.SetUp(false)
			return
		}
		defer resp.Body.Close()

		client.SetUp(resp.StatusCode == http.StatusOK)
	}
}
