package strategy

import (
	"errors"
	"math/rand/v2"
	"sync"
)

// Random is a load balancing strategy that selects a server randomly from the list of eligible servers,
// not accounting for any unit value from the server.
type Random struct {
	// don't think any fields are needed
}

// RandomTest is a reproducible version of Random that uses a seeded random number generator for testing purposes.
// It allows for predictable and repeatable server selection during tests.
type RandomTest struct {
	mu     sync.Mutex
	random *rand.Rand
}

func NewRandom() *Random {
	return &Random{}
}

func NewRandomTest(seed1 uint64, seed2 uint64) *RandomTest {
	return &RandomTest{
		random: rand.New(rand.NewPCG(seed1, seed2)),
	}
}

func (r *Random) Select(eligible []Server) (Server, error) {
	if len(eligible) == 0 {
		return Server{}, errors.New("no eligible servers available")
	}

	// note: IntN is concurrent safe
	i := rand.IntN(len(eligible))
	return eligible[i], nil
}

func (r *RandomTest) Select(eligible []Server) (Server, error) {
	if len(eligible) == 0 {
		return Server{}, errors.New("no eligible servers available")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.random.IntN(len(eligible))
	return eligible[i], nil
}
