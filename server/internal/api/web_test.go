package api_test

import (
	"net/http"
	"strings"
	"testing"
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
		if c.Name == "tend_session" && c.Value != "" {
			foundCookie = true
		}
	}
	if !foundCookie {
		t.Error("expected tend_session cookie in response")
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
