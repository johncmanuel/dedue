package balancer

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

	if !d.HealthCheck(context.Background(), "http://backend.test/app?version=1") {
		t.Fatal("backend with a healthy /health endpoint was marked unhealthy")
	}
}

func TestPeriodicHealthCheckUpdatesBackend(t *testing.T) {
	healthy := &atomic.Bool{}
	healthy.Store(true)
	d := NewDedue()
	d.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		status := http.StatusServiceUnavailable
		if healthy.Load() {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Body: http.NoBody}, nil
	})
	d.AddServer("http://backend.test", 1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.PeriodicHealthCheck(ctx, 5*time.Millisecond)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	waitForHealth := func(want bool) {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		for {
			d.mu.Lock()
			got := d.servers[0].IsHealthy
			d.mu.Unlock()
			if got == want {
				return
			}
			select {
			case <-ticker.C:
			case <-timer.C:
				t.Fatalf("backend health = %t, want %t", got, want)
			}
		}
	}

	healthy.Store(false)
	waitForHealth(false)
	healthy.Store(true)
	waitForHealth(true)
}

func TestPeriodicHealthCheckDoesNotLockDuringProbe(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	calls := &atomic.Int32{}
	d := NewDedue()
	d.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 2 {
			close(started)
			<-release
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	d.AddServer("http://backend.test", 1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.PeriodicHealthCheck(ctx, time.Millisecond)
	}()
	t.Cleanup(func() {
		cancel()
		close(release)
		<-done
	})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("periodic probe did not start")
	}
	if !d.mu.TryLock() {
		t.Fatal("backend mutex remained locked during the probe")
	}
	d.mu.Unlock()
}
