package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sargunv/tend/server/internal/api"
)

func doHealthz(t *testing.T, srv *httptest.Server) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	return resp
}

func TestHealthzOK(t *testing.T) {
	env := setupTestServer(t)

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, env.pool, log)
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	resp := doHealthz(t, srv)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("got status %q, want %q", body["status"], "ok")
	}
}

func TestHealthzDBDown(t *testing.T) {
	env := setupTestServer(t)

	// Close the pool to simulate a dead database.
	env.pool.Close()

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, env.pool, log)
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	resp := doHealthz(t, srv)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "error" {
		t.Fatalf("got status %q, want %q", body["status"], "error")
	}
	if body["error"] != "database unavailable" {
		t.Fatalf("got error %q, want %q — raw DB error may be leaking", body["error"], "database unavailable")
	}
}
