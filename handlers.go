package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func NewProxyHandler(servers []string, balancerType string) *ProxyHandler {
	factory := NewLoadBalancerFactory()
	return &ProxyHandler{
		loadBalancer: factory.CreateLoadBalancer(balancerType, servers),
		httpClient: &defaultHTTPClient{
			Client: &http.Client{Timeout: 30 * time.Second},
		},
	}
}

func NewProxyHandlerWithDeps(loadBalancer LoadBalancer, httpClient HTTPClient) *ProxyHandler {
	return &ProxyHandler{
		loadBalancer: loadBalancer,
		httpClient:   httpClient,
	}
}

func (h *ProxyHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "healthy"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *ProxyHandler) ProxyRequest(w http.ResponseWriter, req *http.Request) {
	serverURL := h.loadBalancer.Next()
	if serverURL == "" {
		writeErrorResponse(w, http.StatusInternalServerError, "no servers configured")
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot read request body")
		return
	}
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	forwardReq, err := http.NewRequest(req.Method, serverURL+req.URL.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot create forwarded request")
		return
	}

	for key, values := range req.Header {
		for _, value := range values {
			forwardReq.Header.Add(key, value)
		}
	}

	if forwardReq.Header.Get("Content-Type") == "" {
		forwardReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.httpClient.Do(forwardReq)
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, fmt.Sprintf("cannot reach %s: %v", serverURL, err))
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
		// Error copying response - client may have disconnected
	}
}
