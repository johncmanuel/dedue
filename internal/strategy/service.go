// Package strategy defines the server selection contract for load balancing.
package strategy

// TODO: strategies to implement:
// 1. Round Robin
// 2. Weighted Round Robin
// 3. Random

// Server is dedue's representation of a backend server.
type Server struct {
	// Address is the address of the server, e.g. "http://localhost:8080"
	Address string
	// Weight is the weight of the server for weighted load balancing. -1 indicates that the server is not used for weighted load balancing.
	// The unit of weight is abstract (i.e. server's processing power, network bandwidth, etc.) and is up to the user to define.
	// The higher the weight, the more requests the server will receive.
	Weight int
	// IsHealthy indicates whether the server is healthy and can receive requests. This is determined by a health check mechanism.
	IsHealthy bool
}

// BalancingStrategy selects a server from the eligible servers.
type BalancingStrategy interface {
	Select(eligible []Server) (Server, error)
}
