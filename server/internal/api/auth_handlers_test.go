package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func TestAuthLoginSuccess(t *testing.T) {
	env := setupTestServer(t)

	// Login doesn't need auth header, but doRequest always adds one. Use a raw request.
	req, _ := http.NewRequestWithContext(t.Context(), "POST", env.Server.URL+"/auth/login", strings.NewReader(`{"email":"test@example.com","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	readJSON(t, resp, &result)
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected a token in response")
	}
	user := result["user"].(map[string]any)
	if user["email"] != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", user["email"])
	}
}

func TestAuthLoginBadPassword(t *testing.T) {
	env := setupTestServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), "POST", env.Server.URL+"/auth/login", strings.NewReader(`{"email":"test@example.com","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestUnauthenticatedRequest(t *testing.T) {
	env := setupTestServer(t)

	// Request without auth header.
	req, _ := http.NewRequestWithContext(t.Context(), "GET", env.Server.URL+"/spaces", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatusClose(t, resp, http.StatusUnauthorized)
}

func TestAuthLoginUnknownEmail(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequestAs(t, env, "", "POST", "/auth/login", `{"email":"nobody@example.com","password":"anything"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestAuthTokenCreate(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"my-token"}`)
	assertStatus(t, resp, http.StatusCreated)

	var result map[string]any
	readJSON(t, resp, &result)
	if result["token"] == nil || result["token"] == "" {
		t.Error("expected token in response")
	}
	authToken := result["authToken"].(map[string]any)
	if authToken["name"] != "my-token" {
		t.Errorf("name = %v, want my-token", authToken["name"])
	}
	if authToken["kind"] != "api" {
		t.Errorf("kind = %v, want api", authToken["kind"])
	}
}

func TestAuthTokenList(t *testing.T) {
	env := setupTestServer(t)

	doRequest(t, env, "POST", "/auth/tokens", `{"name":"token-a"}`)
	doRequest(t, env, "POST", "/auth/tokens", `{"name":"token-b"}`)

	resp := doRequest(t, env, "GET", "/auth/tokens", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	// Should include the setup session token + 2 API tokens = 3.
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
}

func TestAuthTokenDelete(t *testing.T) {
	env := setupTestServer(t)

	// Create a token.
	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"disposable"}`)
	assertStatus(t, resp, http.StatusCreated)
	var result map[string]any
	readJSON(t, resp, &result)
	rawToken := result["token"].(string)
	tokenID := result["authToken"].(map[string]any)["id"].(string)

	// Verify the new token works.
	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusOK)

	// Delete it.
	assertStatusClose(t, doRequest(t, env, "DELETE", "/auth/tokens/"+tokenID, ""), http.StatusNoContent)

	// Using the deleted token should fail.
	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}

func TestAuthTokenDeleteOtherUser(t *testing.T) {
	env := setupTestServer(t)

	// Create a token as owner.
	resp := doRequest(t, env, "POST", "/auth/tokens", `{"name":"owner-token"}`)
	assertStatus(t, resp, http.StatusCreated)
	var result map[string]any
	readJSON(t, resp, &result)
	tokenID := result["authToken"].(map[string]any)["id"].(string)

	// Second user tries to delete owner's token.
	userToken := createTestUser(t, env, "other@example.com", "Other", "pass123")
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/auth/tokens/"+tokenID, ""), http.StatusNotFound)
}

func TestAuthTokenListIsolation(t *testing.T) {
	env := setupTestServer(t)

	// Owner creates a token.
	assertStatusClose(t, doRequest(t, env, "POST", "/auth/tokens", `{"name":"owner-token"}`), http.StatusCreated)

	// Second user creates a token.
	userToken := createTestUser(t, env, "other@example.com", "Other", "pass123")
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/auth/tokens", `{"name":"other-token"}`), http.StatusCreated)

	// Second user lists tokens — should not see owner's tokens.
	resp := doRequestAs(t, env, userToken, "GET", "/auth/tokens", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := page["items"].([]any)
	for _, item := range items {
		tok := item.(map[string]any)
		if tok["name"] == "owner-token" {
			t.Fatal("second user can see owner's token")
		}
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	env := setupTestServer(t)

	// Create a token that's already expired directly in the DB.
	rawToken := "expired-test-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	pastTime := types.EpochSeconds(time.Now().Add(-1 * time.Hour))

	q := dbgen.New(env.db)
	_, err := q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:    1, // owner user
		TokenHash: tokenHash,
		Name:      "expired",
		Kind:      "session",
		ExpiresAt: &pastTime,
		CreatedAt: types.Now(),
	})
	if err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}
