package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/johncmanuel/dedue/internal/balancer"
	"github.com/johncmanuel/dedue/internal/strategy"
)

func TestParseBackendArg(t *testing.T) {
	tests := []struct {
		name       string
		arg        string
		wantAddr   string
		wantWeight int
		wantErr    bool
	}{
		{"default weight", "http://127.0.0.1:9001", "http://127.0.0.1:9001", 1, false},
		{"explicit weight", "http://127.0.0.1:9001,3", "http://127.0.0.1:9001", 3, false},
		{"zero weight", "http://127.0.0.1:9001,0", "", 0, true},
		{"negative weight", "http://127.0.0.1:9001,-1", "", 0, true},
		{"nonnumeric weight", "http://127.0.0.1:9001,high", "", 0, true},
		{"missing weight", "http://127.0.0.1:9001,", "", 0, true},
		{"invalid URL", "http://", "", 0, true},
		{"unsupported scheme", "ftp://127.0.0.1:9001", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			address, weight, err := parseBackendArg(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseBackendArg(%q) error = %v, want error %t", tt.arg, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if address != tt.wantAddr || weight != tt.wantWeight {
				t.Errorf("parseBackendArg(%q) = (%q, %d), want (%q, %d)", tt.arg, address, weight, tt.wantAddr, tt.wantWeight)
			}
		})
	}
}

func mockBackend(t *testing.T, id string, status int) *httptest.Server {
	t.Helper()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", id)
		w.WriteHeader(status)
	}))
	t.Cleanup(s.Close)
	return s
}

func mockHealthyServer(t *testing.T, id string) *httptest.Server {
	t.Helper()
	return mockBackend(t, id, http.StatusOK)
}

func liveBackendResponse(t *testing.T, client *http.Client, address string) (int, string) {
	t.Helper()
	response, err := client.Get(address)
	if err != nil {
		t.Fatalf("GET %s: %v", address, err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	return response.StatusCode, response.Header.Get("X-Backend")
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

// TestRoundRobinSkipsUnhealthyBackends tests that the round robin algorithm skips unhealthy backends and returns the expected responses from healthy backends.
func TestRoundRobinSkipsUnhealthyBackends(t *testing.T) {
	unhealthyFirst := mockBackend(t, "a", http.StatusServiceUnavailable)
	healthyFirst := mockHealthyServer(t, "b")
	unhealthyMiddle := mockBackend(t, "c", http.StatusServiceUnavailable)
	healthyLast := mockHealthyServer(t, "d")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()
	d.AddServer(unhealthyFirst.URL, 1)
	d.AddServer(healthyFirst.URL, 1)
	d.AddServer(unhealthyMiddle.URL, 1)
	d.AddServer(healthyLast.URL, 1)

	for i, want := range []string{"b", "d", "b", "d"} {
		response := httptest.NewRecorder()
		d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, response.Code, http.StatusOK)
		}
		if got := response.Header().Get("X-Backend"); got != want {
			t.Fatalf("request %d: backend = %q, want %q", i+1, got, want)
		}
	}
}

// TestRoundRobinOnehealthyAmongUnhealthyBackends tests that the round robin algorithm correctly selects the only healthy backend among multiple unhealthy backends
// and returns the expected responses.
func TestRoundRobinOneHealthyAmongUnhealthyBackends(t *testing.T) {
	unhealthy := mockBackend(t, "a", http.StatusServiceUnavailable)
	healthy := mockHealthyServer(t, "b")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()
	d.AddServer(unhealthy.URL, 1)
	d.AddServer(healthy.URL, 1)

	for i := range 3 {
		response := httptest.NewRecorder()
		d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, response.Code, http.StatusOK)
		}
		if got := response.Header().Get("X-Backend"); got != "b" {
			t.Fatalf("request %d: backend = %q, want %q", i+1, got, "b")
		}
	}
}

// TestRoundRobinAllUnhealthyBackends tests that the round robin algorithm returns a 503 Service Unavailable status when all backends are unhealthy.
func TestRoundRobinAllUnhealthyBackends(t *testing.T) {
	b1 := mockBackend(t, "a", http.StatusServiceUnavailable)
	b2 := mockBackend(t, "b", http.StatusServiceUnavailable)

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRoundRobin()
	d.AddServer(b1.URL, 1)
	d.AddServer(b2.URL, 1)

	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestRandomSkipsUnhealthyBackends(t *testing.T) {
	unhealthy := mockBackend(t, "a", http.StatusServiceUnavailable)
	healthyFirst := mockHealthyServer(t, "b")
	healthySecond := mockHealthyServer(t, "c")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRandomTest(42, 99)
	d.AddServer(unhealthy.URL, 1)
	d.AddServer(healthyFirst.URL, 1)
	d.AddServer(healthySecond.URL, 1)

	for i, want := range []string{"c", "b", "c", "b", "b", "c", "c", "c"} {
		response := httptest.NewRecorder()
		d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, response.Code, http.StatusOK)
		}
		if got := response.Header().Get("X-Backend"); got != want {
			t.Fatalf("request %d: backend = %q, want %q", i+1, got, want)
		}
	}
}

func TestRandomAllUnhealthyBackends(t *testing.T) {
	unhealthy := mockBackend(t, "a", http.StatusServiceUnavailable)
	d := balancer.NewDedue()
	d.Strategy = strategy.NewRandomTest(42, 99)
	d.AddServer(unhealthy.URL, 1)

	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestRandomNoBackends(t *testing.T) {
	d := balancer.NewDedue()
	d.Strategy = strategy.NewRandomTest(42, 99)

	response := httptest.NewRecorder()
	d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestRandomOneHealthyAmongUnhealthyBackends(t *testing.T) {
	unhealthy := mockBackend(t, "a", http.StatusServiceUnavailable)
	healthy := mockHealthyServer(t, "b")

	d := balancer.NewDedue()
	d.Strategy = strategy.NewRandomTest(42, 99)
	d.AddServer(unhealthy.URL, 1)
	d.AddServer(healthy.URL, 1)

	for i := range 5 {
		response := httptest.NewRecorder()
		d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, response.Code, http.StatusOK)
		}
		if got := response.Header().Get("X-Backend"); got != "b" {
			t.Fatalf("request %d: backend = %q, want %q", i+1, got, "b")
		}
	}
}

func TestRandomConcurrentRequests(t *testing.T) {
	const (
		requests = 32
	)

	first := mockHealthyServer(t, "a")
	second := mockHealthyServer(t, "b")
	d := balancer.NewDedue()
	d.Strategy = strategy.NewRandomTest(42, 99)
	d.AddServer(first.URL, 1)
	d.AddServer(second.URL, 1)

	responses := make(chan *httptest.ResponseRecorder, requests)
	var wg sync.WaitGroup
	for range requests {
		wg.Go(func() {
			response := httptest.NewRecorder()
			d.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			responses <- response
		})
	}
	wg.Wait()
	close(responses)

	counts := make(map[string]int)
	for response := range responses {
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
		counts[response.Header().Get("X-Backend")]++
	}
	if got := counts["a"] + counts["b"]; got != requests {
		t.Errorf("responses from healthy backends = %d, want %d (counts: %v)", got, requests, counts)
	}
	if counts["a"] == 0 || counts["b"] == 0 {
		t.Errorf("both backends should receive requests; counts: %v", counts)
	}
}

func TestWeightedRoundRobinLiveTraffic(t *testing.T) {
	first := mockHealthyServer(t, "a")
	second := mockHealthyServer(t, "b")
	d := balancer.NewDedue()
	d.Strategy = strategy.NewWeightedRoundRobin()
	d.AddServer(first.URL, 3)
	d.AddServer(second.URL, 1)

	frontend := httptest.NewServer(d)
	t.Cleanup(frontend.Close)
	client := frontend.Client()
	for i, want := range []string{"a", "a", "b", "a", "a", "a", "b", "a"} {
		status, backend := liveBackendResponse(t, client, frontend.URL+"/")
		if status != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, status, http.StatusOK)
		}
		if backend != want {
			t.Errorf("request %d: backend = %q, want %q", i+1, backend, want)
		}
	}
}

func TestWeightedRoundRobinLiveSkipsUnhealthyAndDisabled(t *testing.T) {
	unhealthy := mockBackend(t, "unhealthy", http.StatusServiceUnavailable)
	disabled := mockHealthyServer(t, "disabled")
	healthy := mockHealthyServer(t, "healthy")
	d := balancer.NewDedue()
	d.Strategy = &strategy.WeightedRoundRobin{}
	d.AddServer(unhealthy.URL, 10)
	d.AddServer(disabled.URL, 0)
	d.AddServer(healthy.URL, 1)

	frontend := httptest.NewServer(d)
	t.Cleanup(frontend.Close)
	client := frontend.Client()
	for i := range 3 {
		status, backend := liveBackendResponse(t, client, frontend.URL+"/")
		if status != http.StatusOK {
			t.Fatalf("request %d: status = %d, want %d", i+1, status, http.StatusOK)
		}
		if backend != "healthy" {
			t.Errorf("request %d: backend = %q, want %q", i+1, backend, "healthy")
		}
	}
}

func TestWeightedRoundRobinLiveNoPositiveWeight(t *testing.T) {
	backend := mockHealthyServer(t, "disabled")
	d := balancer.NewDedue()
	d.Strategy = strategy.NewWeightedRoundRobin()
	d.AddServer(backend.URL, 0)

	frontend := httptest.NewServer(d)
	t.Cleanup(frontend.Close)
	status, selected := liveBackendResponse(t, frontend.Client(), frontend.URL+"/")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
	if selected != "" {
		t.Errorf("backend = %q, want no backend", selected)
	}
}
