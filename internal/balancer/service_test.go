package balancer

import (
	"net/http"
	"sync"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestConcurrentAddServerKeepsEachHealthResult(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	d := NewDedue()
	d.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		started <- req.URL.Host
		<-release
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	var workers sync.WaitGroup
	workers.Add(2)
	for _, address := range []string{"http://one.test", "http://two.test"} {
		go func() {
			defer workers.Done()
			d.AddServer(address, 1)
		}()
	}
	first := <-started
	second := <-started
	close(release)
	workers.Wait()

	if first == second {
		t.Fatalf("health checks both targeted %q", first)
	}
	d.mu.Lock()
	servers := append([]Server(nil), d.servers...)
	d.mu.Unlock()
	if len(servers) != 2 {
		t.Fatalf("server count = %d, want 2", len(servers))
	}
	for _, server := range servers {
		if !server.IsHealthy {
			t.Errorf("server %q was marked unhealthy", server.Address)
		}
	}
}

func TestHealthCheckUsesHealthEndpoint(t *testing.T) {
	d := NewDedue()
	d.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.URL.Path; got != "/health" {
			t.Errorf("health check path = %q, want /health", got)
			return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	if !d.HealthCheck("http://backend.test/app?version=1") {
		t.Fatal("backend with a healthy /health endpoint was marked unhealthy")
	}
}
