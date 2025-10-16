package main

import (
	"net/http"
	"sync"
	"time"
)

type roundRobinLoadBalancer struct {
	clients      []HTTPClient
	currentIndex int
	mutex        sync.Mutex
}

func (r *roundRobinLoadBalancer) Next() HTTPClient {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.clients) == 0 {
		return nil
	}

	client := r.clients[r.currentIndex]
	r.currentIndex = (r.currentIndex + 1) % len(r.clients)
	return client
}

func newRoundRobinLoadBalancer(servers []string) *roundRobinLoadBalancer {
	clients := make([]HTTPClient, len(servers))
	for i, serverURL := range servers {
		clients[i] = &defaultHTTPClient{
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
			baseURL: serverURL,
		}
	}
	return &roundRobinLoadBalancer{clients: clients}
}
