package main

import "sync"

type roundRobinLoadBalancer struct {
	servers      []string
	currentIndex int
	mutex        sync.Mutex
}

func (r *roundRobinLoadBalancer) Next() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.servers) == 0 {
		return ""
	}

	server := r.servers[r.currentIndex]
	r.currentIndex = (r.currentIndex + 1) % len(r.servers)
	return server
}
