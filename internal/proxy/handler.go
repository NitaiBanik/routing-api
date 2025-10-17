package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"routing-api/internal/loadbalancer"
	"routing-api/internal/logger"

	"go.uber.org/zap"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type ProxyHandler struct {
	clientProvider loadbalancer.ClientProvider
	logger         logger.Logger
}

func NewProxyHandler(clientProvider loadbalancer.ClientProvider, logger logger.Logger) *ProxyHandler {
	return &ProxyHandler{
		clientProvider: clientProvider,
		logger:         logger,
	}
}

func (h *ProxyHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "healthy"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *ProxyHandler) ProxyRequest(w http.ResponseWriter, req *http.Request) {
	log := h.logger

	client := h.clientProvider.GetClient()
	if client == nil {
		log.Error("No servers configured")
		http.Error(w, "no servers configured", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Cannot reach server",
			zap.String("method", req.Method),
			zap.String("path", req.URL.Path),
			zap.Error(err),
		)
		http.Error(w, "cannot reach server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	backendURL := client.GetBaseURL()
	log.Info("ROUTING-API-BACKEND",
		zap.String("method", req.Method),
		zap.String("backend_url", backendURL),
		zap.String("path", req.URL.Path),
		zap.Int("status", resp.StatusCode),
	)

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Error("Error copying response body",
			zap.String("method", req.Method),
			zap.String("path", req.URL.Path),
			zap.Error(err),
		)
	}
}

func (h *ProxyHandler) StartHealthChecks(ctx context.Context, interval time.Duration) {
	h.clientProvider.StartHealthChecks(ctx, interval)
}
