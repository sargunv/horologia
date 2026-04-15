package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sargunv/horologia/server/internal/api"
)

func postLogin(t *testing.T, env *testEnv, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/login",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestLoginSuccess(t *testing.T) {
	env := setupTestServer(t)

	resp := postLogin(t, env, `{"email":"test@example.com","password":"password"}`)
	assertStatus(t, resp, http.StatusOK)

	// Should set a session cookie.
	var foundCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "horologia_session" && c.Value != "" {
			foundCookie = true
		}
	}
	if !foundCookie {
		t.Error("expected horologia_session cookie in response")
	}

	// Should return the user in the body.
	var result map[string]any
	readJSON(t, resp, &result)
	user := jsonAs[map[string]any](t, result["user"])
	if user["email"] != "test@example.com" {
		t.Errorf("email = %v, want test@example.com", user["email"])
	}
}

func TestLoginBadPassword(t *testing.T) {
	env := setupTestServer(t)

	resp := postLogin(t, env, `{"email":"test@example.com","password":"wrong"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestLoginUnknownEmail(t *testing.T) {
	env := setupTestServer(t)

	resp := postLogin(t, env, `{"email":"nobody@example.com","password":"anything"}`)
	assertStatusClose(t, resp, http.StatusBadRequest)
}

func TestLoginRejectsFormContentType(t *testing.T) {
	env := setupTestServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/login",
		strings.NewReader(`email=test@example.com&password=password`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusUnsupportedMediaType)
}

func TestLoginRejectsMissingContentType(t *testing.T) {
	env := setupTestServer(t)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/login",
		strings.NewReader(`{"email":"test@example.com","password":"password"}`))
	// No Content-Type header set.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusBadRequest)
}

func postLogout(t *testing.T, env *testEnv, sessionToken string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/logout", nil)
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "horologia_session", Value: sessionToken})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestLogoutWithSession(t *testing.T) {
	env := setupTestServer(t)

	// Login to get a session cookie.
	loginResp := postLogin(t, env, `{"email":"test@example.com","password":"password"}`)
	assertStatus(t, loginResp, http.StatusOK)
	var sessionToken string
	for _, c := range loginResp.Cookies() {
		if c.Name == "horologia_session" {
			sessionToken = c.Value
		}
	}
	_ = loginResp.Body.Close()
	if sessionToken == "" {
		t.Fatal("no session cookie from login")
	}

	// Verify the session token works.
	assertStatusClose(t, doRequestAs(t, env, sessionToken, "GET", "/users/me", ""), http.StatusOK)

	// Logout.
	logoutResp := postLogout(t, env, sessionToken)
	assertStatusClose(t, logoutResp, http.StatusNoContent)

	// Cookie should be cleared (MaxAge < 0).
	for _, c := range logoutResp.Cookies() {
		if c.Name == "horologia_session" && c.MaxAge >= 0 {
			t.Error("expected horologia_session cookie to be cleared (MaxAge < 0)")
		}
	}

	// The token should no longer work.
	assertStatusClose(t, doRequestAs(t, env, sessionToken, "GET", "/users/me", ""), http.StatusUnauthorized)
}

func TestLogoutWithoutSession(t *testing.T) {
	env := setupTestServer(t)

	resp := postLogout(t, env, "")
	assertStatusClose(t, resp, http.StatusNoContent)
}

func TestLoginDisabledPasswordAuth(t *testing.T) {
	env := setupTestServer(t)

	// Override: create a new server with password auth disabled.
	log := slog.New(slog.DiscardHandler)
	handler := &api.Handler{Pool: env.pool, Log: log, PasswordAuthEnabled: false}
	h, err := api.NewServer(handler, log)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	srv := httptest.NewServer(api.MountWebAuth(h, handler))
	t.Cleanup(srv.Close)

	// POST /auth/login should 403 when password auth is disabled.
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/auth/login",
		strings.NewReader(`{"email":"test@example.com","password":"password"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestAuthConfigPasswordEnabled(t *testing.T) {
	env := setupTestServer(t)

	resp := doRequestAs(t, env, "", "GET", "/auth/config", "")
	assertStatus(t, resp, http.StatusOK)

	var config map[string]any
	readJSON(t, resp, &config)

	password := jsonAs[map[string]any](t, config["password"])
	if password["enabled"] != true {
		t.Errorf("password.enabled = %v, want true", password["enabled"])
	}
}

func TestAuthConfigPasswordDisabled(t *testing.T) {
	env := setupTestServer(t)

	// Create a server with password auth disabled.
	log := slog.New(slog.DiscardHandler)
	handler := &api.Handler{Pool: env.pool, Log: log, PasswordAuthEnabled: false}
	h, err := api.NewServer(handler, log)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	srv := httptest.NewServer(api.MountWebAuth(h, handler))
	t.Cleanup(srv.Close)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/auth/config", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var config map[string]any
	readJSON(t, resp, &config)

	password := jsonAs[map[string]any](t, config["password"])
	if password["enabled"] != false {
		t.Errorf("password.enabled = %v, want false", password["enabled"])
	}
}

func TestWebErrorCodeIsSnakeCase(t *testing.T) {
	env := setupTestServer(t)

	resp := postLogin(t, env, `{"email":"test@example.com","password":"wrong"}`)
	assertStatus(t, resp, http.StatusBadRequest)

	var body map[string]any
	readJSON(t, resp, &body)

	code, ok := body["code"].(string)
	if !ok {
		t.Fatal("response missing \"code\" field")
	}
	if code != "bad_request" {
		t.Errorf("code = %q, want %q (must be snake_case, not Title Case)", code, "bad_request")
	}
}

func TestBearerAuthTakesPrecedenceOverSessionCookie(t *testing.T) {
	env := setupTestServer(t)

	sessionToken := createTestUser(t, env, "cookie-user@example.com", "Cookie User", "password")

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/users/me", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+env.Token)
	req.AddCookie(&http.Cookie{Name: "horologia_session", Value: sessionToken})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var me map[string]any
	readJSON(t, resp, &me)
	if got := me["email"]; got != "test@example.com" {
		t.Fatalf("email = %v, want test@example.com", got)
	}
}
