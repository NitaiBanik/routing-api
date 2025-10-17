package loadbalancer

import (
	"time"
	"routing-api/internal/circuit"
)

type LoadBalancerConfig struct {
	BalancerType    string
	Servers         []string
	RetryConfig     circuit.RetryConfig
	CircuitConfig   circuit.CircuitBreakerConfig
	RequestTimeout  time.Duration
	ConnectTimeout  time.Duration
	SlowThreshold   time.Duration
	MaxSlowCount    int
}

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(config LoadBalancerConfig) LoadBalancer {
	switch config.BalancerType {
	case "round-robin":
		return newRoundRobinLoadBalancer(config)
	default:
		return newRoundRobinLoadBalancer(config)
	}
}
