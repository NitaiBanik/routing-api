package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"routing-api/internal/logger"

	"go.uber.org/zap"
)

type HealthChecker interface {
	Start(ctx context.Context, clients []HTTPClient, interval time.Duration, onHealthChange func())
}

type httpHealthChecker struct {
	checkPath string
	logger    logger.Logger
}

func NewHTTPHealthChecker(logger logger.Logger) *httpHealthChecker {
	return &httpHealthChecker{
		checkPath: "/health",
		logger:    logger,
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
			h.logger.Error("Failed to create health check request",
				zap.String("url", defaultClient.BaseURL+h.checkPath),
				zap.Error(err),
			)
			client.SetUp(false)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := defaultClient.Client.Do(req)
		if err != nil {
			h.logger.Warn("Health check failed",
				zap.String("url", defaultClient.BaseURL+h.checkPath),
				zap.Error(err),
			)
			client.SetUp(false)
			return
		}
		defer resp.Body.Close()

		isHealthy := resp.StatusCode == http.StatusOK
		client.SetUp(isHealthy)
		
		if !isHealthy {
			h.logger.Warn("Health check returned non-OK status",
				zap.String("url", defaultClient.BaseURL+h.checkPath),
				zap.Int("status", resp.StatusCode),
			)
		}
	}
}
