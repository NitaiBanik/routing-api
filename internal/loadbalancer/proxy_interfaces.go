package loadbalancer

import (
	"context"
	"time"

	"routing-api/internal/health"
)

type ClientProvider interface {
	GetClient() health.HTTPClient
	StartHealthChecks(ctx context.Context, interval time.Duration)
}

type LoadBalancer interface {
	Next() health.HTTPClient
	StartHealthChecks(ctx context.Context, interval time.Duration)
}
