package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type Handler struct {
	applicationAPIs []string
	currentIndex    int
	mutex           sync.Mutex
}

func NewHandler(applicationAPIs []string) *Handler {
	return &Handler{
		applicationAPIs: applicationAPIs,
		currentIndex:    0,
	}
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Timestamp time.Time `json:"timestamp"`
}

func (h *Handler) GetNextAPI() string {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if len(h.applicationAPIs) == 0 {
		return ""
	}

	api := h.applicationAPIs[h.currentIndex]
	h.currentIndex = (h.currentIndex + 1) % len(h.applicationAPIs)
	return api
}

func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status:    "healthy",
		Service:   "routing-api",
		Timestamp: time.Now().UTC(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) ProxyRequest(w http.ResponseWriter, req *http.Request) {
	apiURL := h.GetNextAPI()
	if apiURL == "" {
		http.Error(w, "No application APIs configured", http.StatusInternalServerError)
		return
	}

	// Extract request number from body for better tracking
	requestNum := "unknown"
	bodyBytes, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Reset body for forwarding

	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err == nil {
		if num, ok := payload["request_number"]; ok {
			requestNum = fmt.Sprintf("%v", num)
		}
	}

	log.Printf("ROUTING-API: Forwarding request #%s to %s", requestNum, apiURL)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	forwardReq, err := http.NewRequest(req.Method, apiURL+req.URL.Path, bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, "Failed to create forwarded request", http.StatusInternalServerError)
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

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(forwardReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to forward request to %s: %v", apiURL, err), http.StatusBadGateway)
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
