package main

import (
	"net/http"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	http.Error(w, message, statusCode)
}
