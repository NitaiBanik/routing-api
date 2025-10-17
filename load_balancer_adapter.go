package main

import (
	"context"
	"time"
)

type loadBalancerAdapter struct {
	loadBalancer LoadBalancer
}

func NewLoadBalancerAdapter(loadBalancer LoadBalancer) ClientProvider {
	return &loadBalancerAdapter{
		loadBalancer: loadBalancer,
	}
}

func (a *loadBalancerAdapter) GetClient() HTTPClient {
	return a.loadBalancer.Next()
}

func (a *loadBalancerAdapter) StartHealthChecks(ctx context.Context, interval time.Duration) {
	a.loadBalancer.StartHealthChecks(ctx, interval)
}
