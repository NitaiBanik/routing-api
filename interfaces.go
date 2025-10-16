package main

import "net/http"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type LoadBalancer interface {
	Next() string
}

type ProxyHandler struct {
	loadBalancer LoadBalancer
	httpClient   HTTPClient
}
