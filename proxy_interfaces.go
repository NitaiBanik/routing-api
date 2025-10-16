package main

import (
	"context"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	IsUp() bool
	SetUp(isUp bool)
}

type LoadBalancer interface {
	Next() HTTPClient
	StartHealthChecks(ctx context.Context, interval time.Duration)
}

type ProxyHandler struct {
	loadBalancer LoadBalancer
}
