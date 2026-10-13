package strategy

import (
	"errors"
	"sync"
)

// WeightedRoundRobin (WRR) is a load balancing strategy that distributes requests to servers based on their weights.
// Example: given servers A, B, C with weights 3, 2, 1 respectively, out of every 6 requests the algorithm selects the servers in this order:
// A, A, A, B, B, C. This works, but A (and most likely B) will be overloaded with requests every time.
// Rather than implementing the naive version of the algorithm, which priotizies servers with higher weights
// (even if the higher weighted servers are overloaded), this implementation will use a variation of WRR called "smooth" WRR.
// Smooth WRR interleaves the weighted servers, which aims to evenly distribute requests across the servers. Servers maintain a current_weight
// starting at 0. FOr each request, each server's configured weight is added to its current_weight. A server with the highest weight
// is selected, and its current_weight is reduced by the total weight of all servers.
type WeightedRoundRobin struct {
	// mu protects currentWeights.
	mu sync.Mutex
	// currentWeights is a map of server addresses to their current weights.
	currentWeights map[string]int
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
	if w.currentWeights == nil {
		w.currentWeights = make(map[string]int)
	}

	// get total weight of all eligible servers and update their current weights
	totalWeight := 0
	for i := range eligible {
		// skip servers with non-positive weight
		if eligible[i].Weight <= 0 {
			continue
		}
		totalWeight += eligible[i].Weight
		w.currentWeights[eligible[i].Address] += eligible[i].Weight
	}

	// if all eligible servers have non-positive weight, return an error
	if totalWeight <= 0 {
		return Server{}, errors.New("all eligible servers have non-positive weight")
	}

	// get server with highest current weight
	var selected Server
	for i := range eligible {
		// skip servers with non-positive weight
		if eligible[i].Weight <= 0 {
			continue
		}
		if selected.Address == "" || w.currentWeights[eligible[i].Address] > w.currentWeights[selected.Address] {
			selected = eligible[i]
		}
	}

	// reduce the selected server's current weight by the total weight of all servers
	w.currentWeights[selected.Address] -= totalWeight

	return selected, nil
}
