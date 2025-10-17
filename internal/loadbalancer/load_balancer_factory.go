package loadbalancer

import "routing-api/internal/circuit"

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(balancerType string, servers []string, retryConfig circuit.RetryConfig, circuitConfig circuit.CircuitBreakerConfig) LoadBalancer {
	switch balancerType {
	case "round-robin":
		return newRoundRobinLoadBalancer(servers, retryConfig, circuitConfig)
	default:
		return newRoundRobinLoadBalancer(servers, retryConfig, circuitConfig)
	}
}
