package strategy

import (
	"sync"
	"testing"
)

// TestRoundRobinSelect tests the round robin algorithm for selecting servers from a list of eligible servers.
func TestRoundRobinEmptyEligibleServers(t *testing.T) {
	for _, eligible := range [][]Server{nil, {}} {
		selected, err := NewRoundRobin().Select(eligible)
		if err == nil {
			t.Fatal("Select returned no error for an empty eligible list")
		}
		if selected != (Server{}) {
			t.Fatalf("selected = %+v, want zero Server", selected)
		}
	}
}

// TestRoundRobinEligibleListShrinks tests that the round robin algorithm correctly handles the case where
// the eligible list of servers shrinks between selections.
func TestRoundRobinEligibleListShrinks(t *testing.T) {
	roundRobin := NewRoundRobin()
	servers := []Server{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	for range 2 {
		if _, err := roundRobin.Select(servers); err != nil {
			t.Fatalf("Select returned an unexpected error: %v", err)
		}
	}

	selected, err := roundRobin.Select([]Server{{Address: "b"}})
	if err != nil {
		t.Fatalf("Select returned an unexpected error: %v", err)
	}
	if selected.Address != "b" {
		t.Fatalf("selected backend = %q, want %q", selected.Address, "b")
	}
}

// TestRoundRobinConcurrentSelection tests that the round robin algorithm correctly selects servers in a concurrent environment.
func TestRoundRobinConcurrentSelection(t *testing.T) {
	const (
		requests = 300
	)
	eligible := []Server{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	roundRobin := NewRoundRobin()
	results := make(chan string, requests)
	start := make(chan struct{})
	workers := &sync.WaitGroup{}

	for range requests {
		workers.Go(func() {
			<-start
			selected, err := roundRobin.Select(eligible)
			if err != nil {
				t.Errorf("Select returned an unexpected error: %v", err)
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
	for _, server := range eligible {
		if got, want := counts[server.Address], requests/len(eligible); got != want {
			t.Errorf("backend %q selected %d times, want %d", server.Address, got, want)
		}
	}
}
