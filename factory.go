package main

type LoadBalancerFactory struct{}

func NewLoadBalancerFactory() *LoadBalancerFactory {
	return &LoadBalancerFactory{}
}

func (f *LoadBalancerFactory) CreateLoadBalancer(balancerType string, servers []string) LoadBalancer {
	switch balancerType {
	case "round-robin":
		return &roundRobinLoadBalancer{servers: servers}
	default:
		return &roundRobinLoadBalancer{servers: servers} // default to round-robin
	}
}
