package loadbalancer

import (
	"routing-api/internal/circuit"
	"routing-api/internal/logger"
)

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(balancerType string, servers []string, circuitConfig circuit.CircuitBreakerConfig, logger logger.Logger) LoadBalancer {
	switch balancerType {
	case "round-robin":
		return newRoundRobinLoadBalancer(servers, circuitConfig, logger)
	default:
		return newRoundRobinLoadBalancer(servers, circuitConfig, logger)
	}
}
