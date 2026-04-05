package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sargunv/tend/server/internal/mcp"
)

// newMCPServer creates a standalone test HTTP server wrapping only the MCP
// transport at "/mcp". It reuses setupTestServer for the database and token.
func newMCPServer(t *testing.T) (serverURL, token string) {
	t.Helper()
	env := setupTestServer(t)

	// Mount the MCP transport directly — no SPA or API handler needed.
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewTransport(env.pool))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	// Also close the original test server to avoid leaking goroutines.
	t.Cleanup(env.Server.Close)

	return srv.URL, env.Token
}

func mcpInitBody(t *testing.T) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test-client", "version": "0.1.0"},
		},
	})
	if err != nil {
		t.Fatalf("marshal initialize body: %v", err)
	}
	return bytes.NewReader(body)
}

// TestMCPUnauthenticated verifies that requests without a bearer token are rejected.
func TestMCPUnauthenticated(t *testing.T) {
	serverURL, _ := newMCPServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/mcp", mcpInitBody(t))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusUnauthorized)
}

// TestMCPInvalidToken verifies that an invalid bearer token is rejected.
func TestMCPInvalidToken(t *testing.T) {
	serverURL, _ := newMCPServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/mcp", mcpInitBody(t))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer not-a-real-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusUnauthorized)
}

// TestMCPInitializeHandshake verifies that a valid bearer token allows the MCP
// initialize handshake to complete successfully.
func TestMCPInitializeHandshake(t *testing.T) {
	serverURL, token := newMCPServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL+"/mcp", mcpInitBody(t))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]any
	readJSON(t, resp, &result)

	if result["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", result["jsonrpc"])
	}
	if result["error"] != nil {
		t.Errorf("unexpected error in MCP response: %v", result["error"])
	}
	res, ok := result["result"].(map[string]any)
	if !ok {
		t.Fatalf("result field is not an object: %T", result["result"])
	}
	if res["protocolVersion"] == nil {
		t.Error("expected protocolVersion in initialize result")
	}
	serverInfo, ok := res["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo in initialize result")
	}
	if serverInfo["name"] != "Tend" {
		t.Errorf("serverInfo.name = %v, want Tend", serverInfo["name"])
	}
}
