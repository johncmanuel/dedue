package strategy

import (
	"errors"
	"sync"
)

// WeightedRoundRobin is a load balancing strategy that distributes requests to servers based on their weights. It uses a weighted round-robin algorithm to select servers,
// ensuring that servers with higher weights receive more requests over time.
type WeightedRoundRobin struct {
	mu sync.Mutex
}

// NewWeightedRoundRobin creates a new WeightedRoundRobin strategy instance.
func NewWeightedRoundRobin() *WeightedRoundRobin {
	return &WeightedRoundRobin{}
}

func (w *WeightedRoundRobin) Select(eligible []Server) (Server, error) {
	if len(eligible) == 0 {
		return Server{}, errors.New("no eligible servers available")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
}
