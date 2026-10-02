package balancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/johncmanuel/dedue/internal/strategy"
)

// Server is dedue's representation of a backend server.
type Server = strategy.Server

// Dedue is the load balancer
type Dedue struct {
	Client   *http.Client
	Strategy strategy.BalancingStrategy
	mu       sync.Mutex
	servers  []Server
}

// NewDedue creates the load balancer
func NewDedue() *Dedue {
	return &Dedue{
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// ServeHTTP forwards a request to a healthy backend selected by the strategy.
func (d *Dedue) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	healthy := d.HealthCheck(address)

	d.mu.Lock()
	d.servers = append(d.servers, Server{
		Address:   address,
		Weight:    weight,
		IsHealthy: healthy,
	})
	d.mu.Unlock()
}

// HealthCheck checks the health of a server by sending a GET request to its address and checking the response status code.
// If the response status code is 200 OK (or generally 2xx), the server is considered healthy. Otherwise, it's considered unhealthy.
func (d *Dedue) HealthCheck(address string) bool {
	resp, err := d.Client.Get(address)
	if err != nil {
		return false
	}
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	if err := resp.Body.Close(); err != nil {
		return false
	}
	return healthy
}
