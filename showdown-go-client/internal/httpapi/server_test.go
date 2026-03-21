package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPostOnlyRoutesRejectGet(t *testing.T) {
	server := New()

	paths := []string{
		"/api/ping",
		"/api/status",
		"/api/validate-team",
		"/api/mock-battle",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("%s returned %d, want %d", path, rec.Code, http.StatusMethodNotAllowed)
			}
			if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
				t.Fatalf("%s Allow header = %q, want %q", path, allow, http.MethodPost)
			}
		})
	}
}

func TestHealthzReturnsOK(t *testing.T) {
	server := New()
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("healthz returned %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("healthz Content-Type = %q, want application/json", ct)
	}
}

func TestPingRejectsInvalidServerURL(t *testing.T) {
	server := New()

	tests := []struct {
		name string
		body string
	}{
		{"empty url", `{"server_url":"","username":"test"}`},
		{"ftp scheme", `{"server_url":"ftp://example.com","username":"test"}`},
		{"no scheme", `{"server_url":"://bad","username":"test"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/ping", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			server.Handler().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("ping with %s returned %d, want %d", tt.name, rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestPingRejectsInvalidJSON(t *testing.T) {
	server := New()
	req := httptest.NewRequest(http.MethodPost, "/api/ping", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ping with invalid JSON returned %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
