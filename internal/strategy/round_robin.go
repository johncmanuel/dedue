package strategy

import (
	"errors"
	"sync"
)

// RoundRobin is a load balancing strategy that distributes requests in a round-robin fashion. It uses an index to track the next server to be selected from the list of eligible servers.
// Each time a server is selected, the index increments. When reaching the end of the list, it wraps around to the beginning of the list.
type RoundRobin struct {
	// mu is a mutex used to synchronize access to currentIndex.
	mu sync.Mutex
	// currentIndex keeps track of the index of the selected server.
	currentIndex int
}

func (r *RoundRobin) IncrementIndex(eligible []Server) {
	r.currentIndex = (r.currentIndex + 1) % len(eligible)
}

// Select returns the next eligible server.
func (r *RoundRobin) Select(eligible []Server) (Server, error) {
	if len(eligible) == 0 {
		return Server{}, errors.New("no eligible servers available")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// health checks can shrink the eligible list, so need to ensure the currentIndex is within bounds
	// befire accessing a server from the eligible list. basically if the currentIndex is out of bounds,
	// wrap it around to the beginning of the list.
	r.currentIndex %= len(eligible)

	defer r.IncrementIndex(eligible)
	return eligible[r.currentIndex], nil
}

// NewRoundRobin creates a new RoundRobin strategy instance.
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}
