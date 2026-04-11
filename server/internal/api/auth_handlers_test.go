package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func TestUnauthenticatedRequest(t *testing.T) {
	env := setupTestServer(t)

	// Request without auth header.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/spaces", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatusClose(t, resp, http.StatusUnauthorized)
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
	authToken := jsonAs[map[string]any](t, result["authToken"])
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
	items := jsonAs[[]any](t, page["items"])
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
	rawToken := jsonAs[string](t, result["token"])
	tokenID := jsonAs[string](t, jsonAs[map[string]any](t, result["authToken"])["id"])

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
	tokenID := jsonAs[string](t, jsonAs[map[string]any](t, result["authToken"])["id"])

	// Second user tries to delete owner's token.
	userToken := createTestUser(t, env, "other@example.com", "Other", "pass1234")
	assertStatusClose(t, doRequestAs(t, env, userToken, "DELETE", "/auth/tokens/"+tokenID, ""), http.StatusNotFound)
}

func TestAuthTokenListIsolation(t *testing.T) {
	env := setupTestServer(t)

	// Owner creates a token.
	assertStatusClose(t, doRequest(t, env, "POST", "/auth/tokens", `{"name":"owner-token"}`), http.StatusCreated)

	// Second user creates a token.
	userToken := createTestUser(t, env, "other@example.com", "Other", "pass1234")
	assertStatusClose(t, doRequestAs(t, env, userToken, "POST", "/auth/tokens", `{"name":"other-token"}`), http.StatusCreated)

	// Second user lists tokens — should not see owner's tokens.
	resp := doRequestAs(t, env, userToken, "GET", "/auth/tokens", "")
	assertStatus(t, resp, http.StatusOK)
	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	for _, item := range items {
		tok := jsonAs[map[string]any](t, item)
		if tok["name"] == "owner-token" {
			t.Fatal("second user can see owner's token")
		}
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	env := setupTestServer(t)

	// Look up the owner user ID via the standard helper.
	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	// Create a token that's already expired directly in the DB.
	rawToken := "expired-test-token"
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])
	pastTime := time.Now().Add(-1 * time.Hour)

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:    ownerNumericID,
		TokenHash: tokenHash,
		Name:      "expired",
		Kind:      dbgen.AuthTokenKindSession,
		ExpiresAt: pgtype.Timestamptz{Time: pastTime, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("create expired token: %v", err)
	}

	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}

func TestOAuthAccessTokenAccepted(t *testing.T) {
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	rawToken := "oauth-access-test-token" //nolint:gosec // test token fixture
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{ //nolint:gosec // test token fixture
		UserID:        ownerNumericID,
		TokenHash:     tokenHash,
		Name:          "Tend CLI",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "tend-cli", Valid: true},
		OauthScopes:   []string{"profile:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/api", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/users/me", ""), http.StatusOK)
}

func TestAuthTokenListExcludesOAuthTokens(t *testing.T) {
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	q := dbgen.New(env.pool)
	tokenHashBytes := sha256.Sum256([]byte("oauth-access-list-token")) //nolint:gosec // test token fixture
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        ownerNumericID,
		TokenHash:     hex.EncodeToString(tokenHashBytes[:]),
		Name:          "Hidden OAuth Token",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "tend-cli", Valid: true},
		OauthScopes:   []string{"profile:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/api", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	resp := doRequest(t, env, "GET", "/auth/tokens", "")
	assertStatus(t, resp, http.StatusOK)

	var page map[string]any
	readJSON(t, resp, &page)
	items := jsonAs[[]any](t, page["items"])
	for _, item := range items {
		token := jsonAs[map[string]any](t, item)
		if token["name"] == "Hidden OAuth Token" {
			t.Fatal("auth token list exposed oauth-issued token")
		}
	}
}

func TestOAuthAccessTokenMissingScopeForbidden(t *testing.T) {
	env := setupTestServer(t)

	ownerID := getUserID(t, env, env.Token)
	ownerNumericID, err := types.ParseUserID(ownerID)
	if err != nil {
		t.Fatalf("parse owner ID %q: %v", ownerID, err)
	}

	rawToken := "oauth-profile-only-token" //nolint:gosec // test token fixture
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        ownerNumericID,
		TokenHash:     tokenHash,
		Name:          "Tend CLI",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "tend-cli", Valid: true},
		OauthScopes:   []string{"profile:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/api", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	assertStatusClose(t, doRequestAs(t, env, rawToken, "GET", "/spaces", ""), http.StatusForbidden)
}

func TestOAuthAccessTokenStillRequiresOwner(t *testing.T) {
	env := setupTestServer(t)

	userToken := createTestUser(t, env, "worker@example.com", "Worker", "password")
	oauthToken := createOAuthAccessTokenForBearerUser(t, env, userToken, "users:write")

	resp := doRequestAs(t, env, oauthToken, "POST", "/users", `{"email":"new@example.com","name":"New User","password":"password"}`)
	assertStatusClose(t, resp, http.StatusForbidden)
}

func TestOAuthAccessTokenStillRequiresSpaceMembership(t *testing.T) {
	env := setupTestServer(t)

	createSpace(t, env, "secret", "Secret")
	userToken := createTestUser(t, env, "outsider@example.com", "Outsider", "password")
	oauthToken := createOAuthAccessTokenForBearerUser(t, env, userToken, "spaces:read")

	resp := doRequestAs(t, env, oauthToken, "GET", "/spaces/secret", "")
	assertStatusClose(t, resp, http.StatusNotFound)
}

func createOAuthAccessTokenForBearerUser(t *testing.T, env *testEnv, bearerToken string, scopes ...string) string {
	t.Helper()

	userID := getUserID(t, env, bearerToken)
	userNumericID, err := types.ParseUserID(userID)
	if err != nil {
		t.Fatalf("parse user ID %q: %v", userID, err)
	}

	rawToken := fmt.Sprintf("oauth-%s-%s", t.Name(), userID)
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        userNumericID,
		TokenHash:     tokenHash,
		Name:          "Tend CLI",
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "tend-cli", Valid: true},
		OauthScopes:   scopes,
		OauthResource: pgtype.Text{String: env.Server.URL + "/api", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth access token: %v", err)
	}

	return rawToken
}
