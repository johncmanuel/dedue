package strategy

// RoundRobin is a load balancing strategy that distributes requests in a round-robin fashion.
type RoundRobin struct {
	currentIndex int
}

// Select returns the next eligible server.
func (r *RoundRobin) Select(eligible []Server) (Server, error) {
	// TODO
	return eligible[r.currentIndex], nil
}

// NewRoundRobin creates a new RoundRobin strategy instance.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{
		currentIndex: 0,
	}
}
