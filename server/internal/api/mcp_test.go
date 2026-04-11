package api_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/mcp"
	"github.com/sargunv/tend/server/internal/types"
)

// mcpHandler returns an MCP transport wired to a real test database, plus the
// bearer token for the seeded test user. Uses httptest.NewRecorder for direct
// in-process testing — no extra HTTP server needed.
func mcpHandler(t *testing.T) (handler http.Handler, token string) {
	t.Helper()
	env := setupTestServer(t)
	return mcp.NewTransport(env.pool, env.Handler), env.Token
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

// doMCP sends a POST to the MCP handler and returns the recorded response.
func doMCP(t *testing.T, handler http.Handler, token string) *http.Response {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", mcpInitBody(t))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

// TestMCPUnauthenticated verifies that requests without a bearer token are rejected.
func TestMCPUnauthenticated(t *testing.T) {
	handler, _ := mcpHandler(t)
	resp := doMCP(t, handler, "")
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusUnauthorized)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// TestMCPInvalidToken verifies that an invalid bearer token is rejected.
func TestMCPInvalidToken(t *testing.T) {
	handler, _ := mcpHandler(t)
	resp := doMCP(t, handler, "not-a-real-token")
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusUnauthorized)
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// TestMCPExpiredToken verifies that an expired bearer token is rejected.
func TestMCPExpiredToken(t *testing.T) {
	env := setupTestServer(t)
	handler := mcp.NewTransport(env.pool, env.Handler)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	rawToken := "expired-mcp-test-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:    ownerNumericID,
		TokenHash: tokenHash,
		Name:      "expired",
		Kind:      dbgen.AuthTokenKindSession,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(-time.Hour), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	resp := doMCP(t, handler, rawToken)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusUnauthorized)
}

// TestMCPInitializeHandshake verifies that a valid bearer token allows the MCP
// initialize handshake to complete successfully.
func TestMCPInitializeHandshake(t *testing.T) {
	handler, token := mcpHandler(t)
	resp := doMCP(t, handler, token)
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

func TestMCPOAuthAccessTokenInitializeHandshake(t *testing.T) {
	env := setupTestServer(t)
	handler := mcp.NewTransport(env.pool, env.Handler)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	rawToken := "oauth-mcp-test-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        ownerNumericID,
		TokenHash:     tokenHash,
		Name:          "Tend MCP",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "remote-mcp", Valid: true},
		OauthScopes:   []string{"profile:read", "spaces:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/mcp", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	resp := doMCP(t, handler, rawToken)
	defer func() { _ = resp.Body.Close() }()

	assertStatus(t, resp, http.StatusOK)
}
