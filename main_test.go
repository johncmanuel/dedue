package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johncmanuel/dedue/internal/balancer"
	"github.com/johncmanuel/dedue/internal/strategy"
)

// mockHealthyServer creates a mock HTTP server that responds with a 200 OK status and a custom header indicating the backend ID.
// Used for testing the load balancer's behavior with healthy backends.
func mockHealthyServer(t *testing.T, id string) *httptest.Server {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", id)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s.Close)
	return s
}

// TestNoHealthyBackends tests that the load balancer returns a 503 Service Unavailable status when there are no healthy backends available.
func TestNoHealthyBackends(t *testing.T) {
	d := balancer.NewDedue()
	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

// TestNoBalancingStrategy tests that the load balancer returns a 503 Service Unavailable status when no balancing strategy is configured.
func TestNoBalancingStrategy(t *testing.T) {
	b := mockHealthyServer(t, "a")
	d := balancer.NewDedue()
	d.AddServer(b.URL, 1)
	if d.Strategy != nil {
		t.Fatal("new balancer unexpectedly has a strategy")
	}
	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

// TestRoundRobinSingleHealthyBackend tests the round robin algorithm for a healthy backend and returns the expected response.
func TestRoundRobinSingleHealthyBackend(t *testing.T) {
	b := mockHealthyServer(t, "a")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()
	d.AddServer(b.URL, 1)

	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/resource?x=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("X-Backend"); got != "a" {
		t.Fatalf("backend = %q, want %q", got, "a")
	}
}

// TestRoundRobinMultipleHealthyBackends tests the round robin algorithm for multiple healthy backends and returns the expected responses.
func TestRoundRobinMultipleHealthyBackends(t *testing.T) {
	b1 := mockHealthyServer(t, "a")
	b2 := mockHealthyServer(t, "b")
	b3 := mockHealthyServer(t, "c")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()
	d.AddServer(b1.URL, 1)
	d.AddServer(b2.URL, 1)
	d.AddServer(b3.URL, 1)

	for i, want := range []string{"a", "b", "c", "a", "b", "c"} {
		response := httptest.NewRecorder()
		d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/resource?x=1", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, response.Code, http.StatusOK)
		}
		if got := response.Header().Get("X-Backend"); got != want {
			t.Fatalf("request %d: backend = %q, want %q", i+1, got, want)
		}
	}
}
