package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func NewProxyHandler(servers []string, balancerType string) *ProxyHandler {
	loadBalancerFactory := NewLoadBalancerFactory()

	return &ProxyHandler{
		loadBalancer: loadBalancerFactory.CreateLoadBalancer(balancerType, servers),
	}
}

func NewProxyHandlerWithDeps(loadBalancer LoadBalancer) *ProxyHandler {
	return &ProxyHandler{
		loadBalancer: loadBalancer,
	}
}

func (h *ProxyHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "healthy"}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *ProxyHandler) ProxyRequest(w http.ResponseWriter, req *http.Request) {
	client := h.loadBalancer.Next()
	if client == nil {
		writeErrorResponse(w, http.StatusInternalServerError, "no servers configured")
		return
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "cannot read request body")
		return
	}
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	forwardReq, err := http.NewRequest(req.Method, req.URL.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "cannot create forwarded request")
		return
	}

	forwardReq.URL.RawQuery = req.URL.RawQuery

	for key, values := range req.Header {
		for _, value := range values {
			forwardReq.Header.Add(key, value)
		}
	}

	if forwardReq.Header.Get("Content-Type") == "" {
		forwardReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(forwardReq)
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, fmt.Sprintf("cannot reach server: %v", err))
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
	}
}
