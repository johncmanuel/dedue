package strategy

import (
	"sync"
	"testing"
)

func TestWeightedRoundRobinEmptyEligibleServers(t *testing.T) {
	for _, eligible := range [][]Server{nil, {}} {
		selected, err := NewWeightedRoundRobin().Select(eligible)
		if err == nil {
			t.Fatal("Select returned no error for an empty eligible list")
		}
		if selected != (Server{}) {
			t.Fatalf("selected = %+v, want zero Server", selected)
		}
	}
}

func TestWeightedRoundRobinZeroValue(t *testing.T) {
	weighted := &WeightedRoundRobin{}
	for i := range 2 {
		selected, err := weighted.Select([]Server{{Address: "a", Weight: 1}})
		if err != nil {
			t.Fatalf("request %d: Select returned an error: %v", i+1, err)
		}
		if selected.Address != "a" {
			t.Fatalf("request %d: selected %q, want %q", i+1, selected.Address, "a")
		}
	}
}

func TestWeightedRoundRobinSmoothOrder(t *testing.T) {
	weighted := NewWeightedRoundRobin()
	eligible := []Server{{Address: "a", Weight: 3}, {Address: "b", Weight: 1}}

	for i, want := range []string{"a", "a", "b", "a", "a", "a", "b", "a"} {
		// The balancer supplies a fresh eligible slice for each request.
		selected, err := weighted.Select(append([]Server(nil), eligible...))
		if err != nil {
			t.Fatalf("request %d: Select returned an error: %v", i+1, err)
		}
		if selected.Address != want {
			t.Errorf("request %d: selected %q, want %q", i+1, selected.Address, want)
		}
	}
}

func TestWeightedRoundRobinSkipsNonPositiveWeights(t *testing.T) {
	weighted := NewWeightedRoundRobin()
	eligible := []Server{{Address: "disabled", Weight: -1}, {Address: "active", Weight: 1}}

	for i := range 4 {
		selected, err := weighted.Select(eligible)
		if err != nil {
			t.Fatalf("request %d: Select returned an error: %v", i+1, err)
		}
		if selected.Address != "active" {
			t.Errorf("request %d: selected %q, want %q", i+1, selected.Address, "active")
		}
	}
}

func TestWeightedRoundRobinNoPositiveWeights(t *testing.T) {
	selected, err := NewWeightedRoundRobin().Select([]Server{
		{Address: "zero", Weight: 0},
		{Address: "negative", Weight: -1},
	})
	if err == nil {
		t.Fatal("Select returned no error when all weights are nonpositive")
	}
	if selected != (Server{}) {
		t.Fatalf("selected = %+v, want zero Server", selected)
	}
}

func TestWeightedRoundRobinConcurrentSelection(t *testing.T) {
	const (
		requests = 40
	)

	weighted := &WeightedRoundRobin{}
	results := make(chan string, requests)
	start := make(chan struct{})
	workers := &sync.WaitGroup{}

	for range requests {
		workers.Go(func() {
			<-start
			selected, err := weighted.Select([]Server{{Address: "a", Weight: 3}, {Address: "b", Weight: 1}})
			if err != nil {
				t.Errorf("Select returned an error: %v", err)
				return
			}
			results <- selected.Address
		})
	}

	close(start)
	workers.Wait()
	close(results)

	counts := make(map[string]int)
	for address := range results {
		counts[address]++
	}
	if counts["a"] != 30 || counts["b"] != 10 {
		t.Errorf("selection counts = %v, want a:30 and b:10", counts)
	}
}
