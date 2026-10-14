package balancer

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/johncmanuel/dedue/internal/strategy"
)

// Server is dedue's representation of a backend server.
type Server = strategy.Server

// Dedue is the load balancer itself.
type Dedue struct {
	Client *http.Client
	// Strategy is the load balancing strategy used to select a server from the eligible servers.
	Strategy strategy.BalancingStrategy
	// mu is a mutex used to synchronize access to the servers slice.
	mu sync.Mutex
	// servers is a static list of backend servers managed by the load balancer.
	// TODO: implement dynamic server management (add/remove servers at runtime)
	servers []Server
}

// thought to myself: do we really need to use the mutex on data piece?

// NewDedue creates the load balancer
func NewDedue() *Dedue {
	return &Dedue{
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ServeHTTP forwards a request to a healthy backend selected by the strategy.
func (d *Dedue) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check for healthy backends and select one using the configured strategy
	d.mu.Lock()
	eligible := make([]Server, 0, len(d.servers))
	for _, server := range d.servers {
		if server.IsHealthy {
			eligible = append(eligible, server)
		}
	}
	d.mu.Unlock()

	if len(eligible) == 0 {
		http.Error(w, "no healthy backends available", http.StatusServiceUnavailable)
		return
	}

	if d.Strategy == nil {
		http.Error(w, "no balancing strategy configured", http.StatusServiceUnavailable)
		return
	}

	server, err := d.Strategy.Select(eligible)
	if err != nil {
		http.Error(w, "no healthy backends available", http.StatusServiceUnavailable)
		return
	}

	target, err := url.Parse(server.Address)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		http.Error(w, "invalid backend address", http.StatusBadGateway)
		return
	}

	if d.Client == nil {
		http.Error(w, "no HTTP client configured", http.StatusServiceUnavailable)
		return
	}

	// create the reverse proxy and forward the request to the selected backend
	proxy := &httputil.ReverseProxy{
		Transport: d.Client.Transport,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
		},
	}
	proxy.ServeHTTP(w, r)
}

// AddServer adds a server to the load balancer and performs a health check on it.
func (d *Dedue) AddServer(address string, weight int) {
	healthy := d.HealthCheck(context.Background(), address)

	d.mu.Lock()
	d.servers = append(d.servers, Server{
		Address:   address,
		Weight:    weight,
		IsHealthy: healthy,
	})
	d.mu.Unlock()
}

// HealthCheck checks the health of a server by sending a GET request to its /health endpoint.
// If the response status code is 200 OK (or generally 2xx), the server is considered healthy. Otherwise, it's considered unhealthy.
func (d *Dedue) HealthCheck(ctx context.Context, address string) bool {
	probeURL, err := url.Parse(address)
	if err != nil {
		return false
	}

	// ensure the health check is performed on the /health endpoint
	probeURL.Path = "/health"
	probeURL.RawPath = ""
	probeURL.RawQuery = ""
	probeURL.Fragment = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL.String(), nil)
	if err != nil {
		return false
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return false
	}

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	if err := resp.Body.Close(); err != nil {
		return false
	}

	return healthy
}

// PeriodicHealthCheck continuously updates backend health until ctx is canceled.
func (d *Dedue) PeriodicHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// run health checks in a loop until the provided context is canceled
	for {
		// wait for the next tick or context cancellation
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		// add copy of servers to avoid holding the lock while performing health checks
		d.mu.Lock()
		servers := append([]Server(nil), d.servers...)
		d.mu.Unlock()

		// perform health checks for each server
		for i, server := range servers {
			probeCtx, cancel := context.WithTimeout(ctx, time.Second)
			healthy := d.HealthCheck(probeCtx, server.Address)

			cancel()
			if ctx.Err() != nil {
				return
			}

			d.mu.Lock()
			d.servers[i].IsHealthy = healthy
			d.mu.Unlock()
		}
	}
}
