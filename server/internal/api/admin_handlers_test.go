package api_test

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

func TestUsersGet(t *testing.T) {
	env := setupTestServer(t)

	t.Run("get existing user", func(t *testing.T) {
		ownerID := getUserID(t, env, env.Token)
		resp := doRequest(t, env, "GET", "/users/"+ownerID, "")
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["id"] != ownerID {
			t.Errorf("id = %v, want %s", user["id"], ownerID)
		}
		if user["email"] != "test@example.com" {
			t.Errorf("email = %v, want test@example.com", user["email"])
		}
		// hasPassword should be true for the test owner (created with password).
		if user["hasPassword"] != true {
			t.Errorf("hasPassword = %v, want true", user["hasPassword"])
		}
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		resp := doRequest(t, env, "GET", "/users/U999999", "")
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		ownerID := getUserID(t, env, env.Token)
		resp := doRequestAs(t, env, "", "GET", "/users/"+ownerID, "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})

	t.Run("non-owner can get user", func(t *testing.T) {
		token := createTestUser(t, env, "getter@example.com", "Getter", "password")
		ownerID := getUserID(t, env, env.Token)
		resp := doRequestAs(t, env, token, "GET", "/users/"+ownerID, "")
		assertStatusClose(t, resp, http.StatusOK)
	})
}

func TestUsersList(t *testing.T) {
	env := setupTestServer(t)

	t.Run("authenticated user can list users", func(t *testing.T) {
		resp := doRequest(t, env, "GET", "/users", "")
		assertStatus(t, resp, http.StatusOK)

		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		if len(items) < 1 {
			t.Fatalf("expected at least 1 user, got %d", len(items))
		}
		// Verify response shape includes expected fields.
		first := jsonAs[map[string]any](t, items[0])
		for _, field := range []string{"id", "email", "name", "isOwner", "createdAt", "updatedAt"} {
			if _, ok := first[field]; !ok {
				t.Errorf("missing field %q in user list response", field)
			}
		}
	})

	t.Run("non-owner can list users", func(t *testing.T) {
		token := createTestUser(t, env, "viewer@example.com", "Viewer", "password")
		resp := doRequestAs(t, env, token, "GET", "/users", "")
		assertStatus(t, resp, http.StatusOK)

		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		if len(items) < 2 {
			t.Fatalf("expected at least 2 users, got %d", len(items))
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doRequestAs(t, env, "", "GET", "/users", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})
}

func TestUsersCreate(t *testing.T) {
	env := setupTestServer(t)

	t.Run("owner can create user with password", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Alice","email":"alice@example.com","password":"securepassword123"}`)
		assertStatus(t, resp, http.StatusCreated)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["name"] != "Alice" {
			t.Errorf("name = %v, want Alice", user["name"])
		}
		if user["email"] != "alice@example.com" {
			t.Errorf("email = %v, want alice@example.com", user["email"])
		}
		if user["isOwner"] != false {
			t.Errorf("isOwner = %v, want false", user["isOwner"])
		}
	})

	t.Run("owner can create user without password", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Bob","email":"bob@example.com"}`)
		assertStatus(t, resp, http.StatusCreated)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["name"] != "Bob" {
			t.Errorf("name = %v, want Bob", user["name"])
		}
	})

	t.Run("owner can create owner user", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Charlie","email":"charlie@example.com","password":"securepassword123","isOwner":true}`)
		assertStatus(t, resp, http.StatusCreated)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["isOwner"] != true {
			t.Errorf("isOwner = %v, want true", user["isOwner"])
		}
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Alice Dup","email":"alice@example.com","password":"securepassword123"}`)
		assertStatusClose(t, resp, http.StatusConflict)
	})

	t.Run("non-owner gets 403", func(t *testing.T) {
		token := createTestUser(t, env, "regular@example.com", "Regular", "password")
		resp := doRequestAs(t, env, token, "POST", "/users",
			`{"name":"Denied","email":"denied@example.com","password":"securepassword123"}`)
		assertStatusClose(t, resp, http.StatusForbidden)
	})

	t.Run("short password returns 400", func(t *testing.T) {
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Short","email":"short@example.com","password":"abc"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})
}

func TestUsersUpdate(t *testing.T) {
	env := setupTestServer(t)

	// Create a user to update.
	resp := doRequest(t, env, "POST", "/users",
		`{"name":"Update Me","email":"update@example.com","password":"securepassword123"}`)
	assertStatus(t, resp, http.StatusCreated)
	var created map[string]any
	readJSON(t, resp, &created)
	userID := jsonAs[string](t, created["id"])

	t.Run("update name", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"name":"Updated Name"}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["name"] != "Updated Name" {
			t.Errorf("name = %v, want Updated Name", user["name"])
		}
		if user["email"] != "update@example.com" {
			t.Errorf("email changed unexpectedly: %v", user["email"])
		}
	})

	t.Run("update email", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"email":"newemail@example.com"}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["email"] != "newemail@example.com" {
			t.Errorf("email = %v, want newemail@example.com", user["email"])
		}
	})

	t.Run("promote to owner", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"isOwner":true}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["isOwner"] != true {
			t.Errorf("isOwner = %v, want true", user["isOwner"])
		}
	})

	t.Run("demote from owner succeeds when not last", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"isOwner":false}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["isOwner"] != false {
			t.Errorf("isOwner = %v, want false", user["isOwner"])
		}
	})

	t.Run("demote last owner returns 400", func(t *testing.T) {
		// The test env owner is the only owner now.
		ownerID := getUserID(t, env, env.Token)
		resp := doRequest(t, env, "PATCH", "/users/"+ownerID,
			`{"isOwner":false}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("set password then login", func(t *testing.T) {
		// Set a new password via admin PATCH.
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"setPassword":"brandnewpass123"}`)
		assertStatusClose(t, resp, http.StatusOK)

		// Verify the user can log in with the new password.
		resp = doRequestAs(t, env, "", "POST", "/app/auth/login",
			`{"email":"newemail@example.com","password":"brandnewpass123"}`)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("login with new password: got %d; body: %s", resp.StatusCode, body)
		}
		_ = resp.Body.Close()
	})

	t.Run("clear password then login rejected", func(t *testing.T) {
		// Clear the password.
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"clearPassword":true}`)
		assertStatusClose(t, resp, http.StatusOK)

		// Verify login is rejected.
		resp = doRequestAs(t, env, "", "POST", "/app/auth/login",
			`{"email":"newemail@example.com","password":"brandnewpass123"}`)
		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			t.Fatal("login should be rejected after clearing password")
		}
		_ = resp.Body.Close()
	})

	t.Run("set short password via PATCH returns 400", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"setPassword":"abc"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("empty PATCH body returns 200", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID, `{}`)
		assertStatusClose(t, resp, http.StatusOK)
	})

	t.Run("set and clear password simultaneously returns 400", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/"+userID,
			`{"setPassword":"newpass123456","clearPassword":true}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("non-owner cannot update other user", func(t *testing.T) {
		token := createTestUser(t, env, "nonadmin@example.com", "NonAdmin", "password")
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+userID,
			`{"name":"Hacked"}`)
		assertStatusClose(t, resp, http.StatusForbidden)
	})

	t.Run("non-owner can update own name", func(t *testing.T) {
		token := createTestUser(t, env, "selfupdate@example.com", "SelfUpdate", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+selfID,
			`{"name":"New Name"}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["name"] != "New Name" {
			t.Errorf("name = %v, want New Name", user["name"])
		}
	})

	t.Run("non-owner can update own email", func(t *testing.T) {
		token := createTestUser(t, env, "selfemail@example.com", "SelfEmail", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+selfID,
			`{"email":"selfemail-new@example.com"}`)
		assertStatus(t, resp, http.StatusOK)

		var user map[string]any
		readJSON(t, resp, &user)
		if user["email"] != "selfemail-new@example.com" {
			t.Errorf("email = %v, want selfemail-new@example.com", user["email"])
		}
	})

	t.Run("non-owner cannot set isOwner on self", func(t *testing.T) {
		token := createTestUser(t, env, "selfowner@example.com", "SelfOwner", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+selfID,
			`{"isOwner":true}`)
		assertStatusClose(t, resp, http.StatusForbidden)
	})

	t.Run("non-owner can set own password", func(t *testing.T) {
		token := createTestUser(t, env, "selfpwd@example.com", "SelfPwd", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+selfID,
			`{"setPassword":"newpassword123"}`)
		assertStatusClose(t, resp, http.StatusOK)

		// Verify login with new password works.
		resp = doRequestAs(t, env, "", "POST", "/app/auth/login",
			`{"email":"selfpwd@example.com","password":"newpassword123"}`)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("login with new password: got %d; body: %s", resp.StatusCode, body)
		}
		_ = resp.Body.Close()
	})

	t.Run("non-owner can clear own password", func(t *testing.T) {
		token := createTestUser(t, env, "selfclear@example.com", "SelfClear", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "PATCH", "/users/"+selfID,
			`{"clearPassword":true}`)
		assertStatusClose(t, resp, http.StatusOK)

		// Verify login is rejected.
		resp = doRequestAs(t, env, "", "POST", "/app/auth/login",
			`{"email":"selfclear@example.com","password":"password"}`)
		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			t.Fatal("login should be rejected after clearing password")
		}
		_ = resp.Body.Close()
	})

	t.Run("password change revokes other credentials but keeps current session", func(t *testing.T) {
		// Create a user and log in twice to get two session tokens.
		token1 := createTestUser(t, env, "sessiontest@example.com", "SessionTest", "password")
		selfID := getUserID(t, env, token1)

		// Log in again to get a second session.
		resp := doRequestAs(t, env, "", "POST", "/app/auth/login",
			`{"email":"sessiontest@example.com","password":"password"}`)
		assertStatus(t, resp, http.StatusOK)
		var token2 string
		for _, c := range resp.Cookies() {
			if c.Name == "horologia_session" {
				token2 = c.Value
			}
		}
		_ = resp.Body.Close()
		if token2 == "" {
			t.Fatal("second login missing horologia_session cookie")
		}

		// Create API and OAuth credentials for the same user.
		resp = doRequestAs(t, env, token1, "POST", "/auth/tokens", `{"name":"CLI"}`)
		assertStatus(t, resp, http.StatusCreated)
		var apiTokenBody map[string]any
		readJSON(t, resp, &apiTokenBody)
		apiToken := jsonAs[string](t, apiTokenBody["token"])

		oauthAccessToken := createOAuthAccessTokenForBearerUser(t, env, token1, "profile:read")
		oauthRefreshToken := createOAuthRefreshTokenForBearerUser(t, env, token1)

		// All credentials should work before the password change.
		resp = doRequestAs(t, env, token1, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusOK)
		resp = doRequestAs(t, env, token2, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusOK)
		resp = doRequestAs(t, env, apiToken, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusOK)
		resp = doRequestAs(t, env, oauthAccessToken, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusOK)

		// Change password using token1.
		resp = doRequestAs(t, env, token1, "PATCH", "/users/"+selfID,
			`{"setPassword":"changedpass123"}`)
		assertStatusClose(t, resp, http.StatusOK)

		// token1 (current session) should still work.
		resp = doRequestAs(t, env, token1, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusOK)

		// Every other credential should be revoked.
		resp = doRequestAs(t, env, token2, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
		resp = doRequestAs(t, env, apiToken, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
		resp = doRequestAs(t, env, oauthAccessToken, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
		resp = doRequestAs(t, env, oauthRefreshToken, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		resp := doRequest(t, env, "PATCH", "/users/U999999",
			`{"name":"Ghost"}`)
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doRequestAs(t, env, "", "PATCH", "/users/"+userID,
			`{"name":"Anon"}`)
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})
}

func createOAuthRefreshTokenForBearerUser(t *testing.T, env *testEnv, bearerToken string) string {
	t.Helper()

	userID := getUserID(t, env, bearerToken)
	userNumericID, err := types.ParseUserID(userID)
	if err != nil {
		t.Fatalf("parse user ID %q: %v", userID, err)
	}

	rawToken := "oauth-refresh-" + userID
	hash := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(hash[:])

	q := dbgen.New(env.pool)
	_, err = q.CreateAuthToken(t.Context(), dbgen.CreateAuthTokenParams{
		UserID:        userNumericID,
		TokenHash:     tokenHash,
		Name:          "Horologia CLI",
		Kind:          dbgen.AuthTokenKindOauthRefresh,
		ExpiresAt:     pgtype.Timestamptz{Time: time.Now().Add(time.Hour), Valid: true},
		CreatedAt:     pgtype.Timestamptz{Time: time.Now(), Valid: true},
		OauthClientID: pgtype.Text{String: "horologia-cli", Valid: true},
		OauthScopes:   []string{"profile:read"},
		OauthResource: pgtype.Text{String: env.Server.URL + "/api", Valid: true},
	})
	if err != nil {
		t.Fatalf("create oauth refresh token: %v", err)
	}

	return rawToken
}

func TestUsersDelete(t *testing.T) {
	env := setupTestServer(t)

	t.Run("owner can delete user", func(t *testing.T) {
		// Create a user to delete.
		resp := doRequest(t, env, "POST", "/users",
			`{"name":"Deletable","email":"delete@example.com","password":"securepassword123"}`)
		assertStatus(t, resp, http.StatusCreated)
		var created map[string]any
		readJSON(t, resp, &created)
		userID := jsonAs[string](t, created["id"])

		resp = doRequest(t, env, "DELETE", "/users/"+userID, "")
		assertStatusClose(t, resp, http.StatusNoContent)

		// Verify user is gone.
		resp = doRequest(t, env, "GET", "/users", "")
		assertStatus(t, resp, http.StatusOK)
		var result map[string]any
		readJSON(t, resp, &result)
		items := jsonAs[[]any](t, result["items"])
		for _, item := range items {
			u := jsonAs[map[string]any](t, item)
			if u["id"] == userID {
				t.Errorf("deleted user %s still in list", userID)
			}
		}
	})

	t.Run("cannot delete last owner", func(t *testing.T) {
		ownerID := getUserID(t, env, env.Token)
		resp := doRequest(t, env, "DELETE", "/users/"+ownerID, "")
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("non-owner can delete self", func(t *testing.T) {
		token := createTestUser(t, env, "selfdelete@example.com", "SelfDelete", "password")
		selfID := getUserID(t, env, token)
		resp := doRequestAs(t, env, token, "DELETE", "/users/"+selfID, "")
		assertStatusClose(t, resp, http.StatusNoContent)

		// Their token should no longer work.
		resp = doRequestAs(t, env, token, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})

	t.Run("non-owner cannot delete other user", func(t *testing.T) {
		token := createTestUser(t, env, "regular2@example.com", "Regular2", "password")
		ownerID := getUserID(t, env, env.Token)
		resp := doRequestAs(t, env, token, "DELETE", "/users/"+ownerID, "")
		assertStatusClose(t, resp, http.StatusForbidden)
	})

	t.Run("nonexistent user returns 404", func(t *testing.T) {
		resp := doRequest(t, env, "DELETE", "/users/U999999", "")
		assertStatusClose(t, resp, http.StatusNotFound)
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doRequestAs(t, env, "", "DELETE", "/users/U999999", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})

	t.Run("delete cascades auth tokens", func(t *testing.T) {
		// Create a user with a session token.
		token := createTestUser(t, env, "cascade@example.com", "Cascade", "password")
		userID := getUserID(t, env, token)

		// Delete the user.
		resp := doRequest(t, env, "DELETE", "/users/"+userID, "")
		assertStatusClose(t, resp, http.StatusNoContent)

		// Their token should no longer work.
		resp = doRequestAs(t, env, token, "GET", "/users/me", "")
		assertStatusClose(t, resp, http.StatusUnauthorized)
	})

	t.Run("delete cascades space memberships", func(t *testing.T) {
		// Create a space.
		createSpace(t, env, "cascade-space", "Cascade Space")

		// Create a user and add them to the space.
		token := createTestUser(t, env, "spacemember@example.com", "SpaceMember", "password")
		memberID := getUserID(t, env, token)
		assertStatusClose(t, doRequest(t, env, "POST", "/spaces/cascade-space/members",
			`{"userId":"`+memberID+`","role":"member"}`), http.StatusCreated)

		// Verify they are a member.
		resp := doRequest(t, env, "GET", "/spaces/cascade-space/members", "")
		assertStatus(t, resp, http.StatusOK)
		var members map[string]any
		readJSON(t, resp, &members)
		items := jsonAs[[]any](t, members["items"])
		found := false
		for _, item := range items {
			m := jsonAs[map[string]any](t, item)
			if m["userId"] == memberID {
				found = true
			}
		}
		if !found {
			t.Fatal("user should be a member before deletion")
		}

		// Delete the user.
		resp = doRequest(t, env, "DELETE", "/users/"+memberID, "")
		assertStatusClose(t, resp, http.StatusNoContent)

		// Verify they are no longer a member.
		resp = doRequest(t, env, "GET", "/spaces/cascade-space/members", "")
		assertStatus(t, resp, http.StatusOK)
		readJSON(t, resp, &members)
		items = jsonAs[[]any](t, members["items"])
		for _, item := range items {
			m := jsonAs[map[string]any](t, item)
			if m["userId"] == memberID {
				t.Errorf("deleted user %s still in space members", memberID)
			}
		}
	})
}
