package main

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(balancerType string, servers []string, retryConfig RetryConfig, circuitConfig CircuitBreakerConfig) LoadBalancer {
	switch balancerType {
	case "round-robin":
		return newRoundRobinLoadBalancer(servers, retryConfig, circuitConfig)
	default:
		return newRoundRobinLoadBalancer(servers, retryConfig, circuitConfig) // default to round-robin
	}
}
