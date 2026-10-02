package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := newHandler("backend-a")

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}
	if health.Body.Len() != 0 {
		t.Fatalf("health body = %q, want empty", health.Body.String())
	}

	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := root.Header().Get("X-Backend"); got != "backend-a" {
		t.Fatalf("root backend = %q, want %q", got, "backend-a")
	}
}
