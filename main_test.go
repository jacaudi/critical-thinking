package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jacaudi/rubber-ducky-mcp/internal/thinking"
)

func TestCORSDefaultRejectsBrowser(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "https://app.example,https://other.example")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Origin", "https://app.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example" {
		t.Errorf("Allow-Origin = %q, want https://app.example", got)
	}
	if got := rec.Header().Get("Access-Control-Expose-Headers"); got != "mcp-session-id" {
		t.Errorf("Expose-Headers = %q, want mcp-session-id", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
}

func TestCORSAllowsNoOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "")
	h := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	// no Origin header
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for no-origin request, got %d", rec.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	registry := newSessionRegistry()
	registry.add(thinking.NewServer())
	registry.add(thinking.NewServer())

	h := makeHealthHandler(registry)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var body struct {
		Status         string `json:"status"`
		Transport      string `json:"transport"`
		ActiveSessions int    `json:"activeSessions"`
		Version        string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	if body.Transport != "streamable-http" {
		t.Errorf("transport = %q, want streamable-http", body.Transport)
	}
	if body.ActiveSessions != 2 {
		t.Errorf("activeSessions = %d, want 2", body.ActiveSessions)
	}
	// version may be "dev" or whatever -ldflags set; just confirm non-empty.
	if body.Version == "" {
		t.Errorf("version is empty")
	}
}
