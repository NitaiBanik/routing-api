package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func NewProxyHandler(clientProvider ClientProvider) *ProxyHandler {
	return &ProxyHandler{
		clientProvider: clientProvider,
	}
}

func (h *ProxyHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "healthy"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *ProxyHandler) ProxyRequest(w http.ResponseWriter, req *http.Request) {
	client := h.clientProvider.GetClient()
	if client == nil {
		http.Error(w, "no servers configured", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot reach server: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}

func (h *ProxyHandler) StartHealthChecks(ctx context.Context, interval time.Duration) {
	h.clientProvider.StartHealthChecks(ctx, interval)
}
