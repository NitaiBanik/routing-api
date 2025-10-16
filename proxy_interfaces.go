package main

import "net/http"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type LoadBalancer interface {
	Next() HTTPClient
}

type ProxyHandler struct {
	loadBalancer LoadBalancer
}
