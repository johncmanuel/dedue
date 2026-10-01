package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/johncmanuel/dedue/internal/balancer"
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
	d := balancer.NewDedue()
	d.Strategy = nil // explicitly set to nil
	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

// TestProxyForwardsToHealthyBackend tests that the load balancer correctly forwards requests to a healthy backend and returns the expected response.
func TestProxyForwardsToHealthyBackend(t *testing.T) {
	b := mockHealthyServer(t, "a")

	d := balancer.NewDedue()
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

// TestProxyForwardsToHealthyBackends tests that the load balancer correctly forwards requests to multiple healthy backends and return the expected responses.
func TestProxyForwardsToHealthyBackends(t *testing.T) {
	b1 := mockHealthyServer(t, "a")
	b2 := mockHealthyServer(t, "b")
	b3 := mockHealthyServer(t, "c")

	d := balancer.NewDedue()
	d.AddServer(b1.URL, 1)
	d.AddServer(b2.URL, 1)
	d.AddServer(b3.URL, 1)

	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/resource?x=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("X-Backend"); got != "a" {
		t.Fatalf("backend = %q, want %q", got, "a")
	}
}
