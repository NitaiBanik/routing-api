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
	checkPath        string
	failureThreshold int
	logger           logger.Logger
	failureCounts    map[string]int
	mutex            sync.RWMutex
}

func NewHTTPHealthChecker(logger logger.Logger) *httpHealthChecker {
	return &httpHealthChecker{
		checkPath:        "/health",
		failureThreshold: 3,
		logger:           logger,
		failureCounts:    make(map[string]int),
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
	clientURL := client.GetBaseURL()

	req, err := http.NewRequest("GET", clientURL+h.checkPath, nil)
	if err != nil {
		h.logger.Error("Failed to create health check request",
			zap.String("url", clientURL+h.checkPath),
			zap.Error(err),
		)
		h.recordFailure(clientURL, client)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	var resp *http.Response
	if defaultClient, ok := client.(*DefaultHTTPClient); ok {
		resp, err = defaultClient.Client.Do(req)
	} else {
		resp, err = client.Do(req)
	}

	if err != nil {
		h.logger.Warn("Health check failed",
			zap.String("url", clientURL+h.checkPath),
			zap.Error(err),
		)
		h.recordFailure(clientURL, client)
		return
	}
	defer resp.Body.Close()

	isHealthy := resp.StatusCode == http.StatusOK
	if isHealthy {
		h.recordSuccess(clientURL, client)
	} else {
		h.logger.Warn("Health check returned non-OK status",
			zap.String("url", clientURL+h.checkPath),
			zap.Int("status", resp.StatusCode),
		)
		h.recordFailure(clientURL, client)
	}
}

func (h *httpHealthChecker) recordSuccess(clientURL string, client HTTPClient) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.failureCounts[clientURL] = 0

	if !client.IsUp() {
		client.SetUp(true)
		h.logger.Info("Server marked as healthy",
			zap.String("url", clientURL),
		)
	}
}

func (h *httpHealthChecker) recordFailure(clientURL string, client HTTPClient) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.failureCounts[clientURL]++
	failureCount := h.failureCounts[clientURL]

	h.logger.Debug("Health check failure recorded",
		zap.String("url", clientURL),
		zap.Int("consecutive_failures", failureCount),
		zap.Int("threshold", h.failureThreshold),
	)

	if failureCount >= h.failureThreshold && client.IsUp() {
		client.SetUp(false)
		h.logger.Warn("Server marked as unhealthy",
			zap.String("url", clientURL),
			zap.Int("consecutive_failures", failureCount),
			zap.Int("threshold", h.failureThreshold),
		)
	}
}
