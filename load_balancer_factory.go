package main

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(balancerType string, servers []string) LoadBalancer {
	switch balancerType {
	case "round-robin":
		return newRoundRobinLoadBalancer(servers)
	default:
		return newRoundRobinLoadBalancer(servers) // default to round-robin
	}
}
