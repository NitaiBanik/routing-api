package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	// start the routing api with some fake backend urls
	cmd := exec.Command("go", "run", "main.go")
	cmd.Env = append(os.Environ(),
		"API_1=http://localhost:9991",
		"API_2=http://localhost:9992",
		"API_3=http://localhost:9993",
		"PORT=3001",
	)
	cmd.Start()
	defer cmd.Process.Kill()

	time.Sleep(2 * time.Second)

	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "returns 502 when backends are down",
			body:           map[string]interface{}{"test": true},
			expectedStatus: 502,
		},
		{
			name:           "handles complex json with 502",
			body:           map[string]interface{}{"user": map[string]interface{}{"id": 123, "name": "Test"}, "items": []int{1, 2, 3}},
			expectedStatus: 502,
		},
		{
			name:           "empty json returns 502",
			body:           map[string]interface{}{},
			expectedStatus: 502,
		},
		{
			name:           "simple message returns 502",
			body:           map[string]interface{}{"message": "Hello World"},
			expectedStatus: 502,
		},
	}

	client := &http.Client{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest("POST", "http://localhost:3001/testapi", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
