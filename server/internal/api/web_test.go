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
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/app/auth/login",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func postLogout(t *testing.T, env *testEnv, sessionToken string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/app/auth/logout", nil)
	if sessionToken != "" {
		req.AddCookie(&http.Cookie{Name: "horologia_session", Value: sessionToken})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestWebAuthHandlers(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	t.Run("login success", func(t *testing.T) {
		resp := postLogin(t, env, `{"email":"test@example.com","password":"password"}`)
		assertStatus(t, resp, http.StatusOK)

		var foundCookie bool
		for _, c := range resp.Cookies() {
			if c.Name == "horologia_session" && c.Value != "" {
				foundCookie = true
			}
		}
		if !foundCookie {
			t.Error("expected horologia_session cookie in response")
		}

		var result map[string]any
		readJSON(t, resp, &result)
		user := jsonAs[map[string]any](t, result["user"])
		if user["email"] != "test@example.com" {
			t.Errorf("email = %v, want test@example.com", user["email"])
		}
	})

	t.Run("login bad password", func(t *testing.T) {
		resp := postLogin(t, env, `{"email":"test@example.com","password":"wrong"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("login unknown email", func(t *testing.T) {
		resp := postLogin(t, env, `{"email":"nobody@example.com","password":"anything"}`)
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("login rejects form content type", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/app/auth/login",
			strings.NewReader(`email=test@example.com&password=password`))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		assertStatusClose(t, resp, http.StatusUnsupportedMediaType)
	})

	t.Run("login rejects missing content type", func(t *testing.T) {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/app/auth/login",
			strings.NewReader(`{"email":"test@example.com","password":"password"}`))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		assertStatusClose(t, resp, http.StatusBadRequest)
	})

	t.Run("logout with session", func(t *testing.T) {
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

		assertStatusClose(t, doRequestAs(t, env, sessionToken, "GET", "/users/me", ""), http.StatusOK)

		logoutResp := postLogout(t, env, sessionToken)
		for _, c := range logoutResp.Cookies() {
			if c.Name == "horologia_session" && c.MaxAge >= 0 {
				t.Error("expected horologia_session cookie to be cleared (MaxAge < 0)")
			}
		}
		assertStatusClose(t, logoutResp, http.StatusNoContent)
		assertStatusClose(t, doRequestAs(t, env, sessionToken, "GET", "/users/me", ""), http.StatusUnauthorized)
	})

	t.Run("logout without session", func(t *testing.T) {
		resp := postLogout(t, env, "")
		assertStatusClose(t, resp, http.StatusNoContent)
	})

	t.Run("web error code is snake case", func(t *testing.T) {
		resp := postLogin(t, env, `{"email":"test@example.com","password":"wrong"}`)
		assertStatus(t, resp, http.StatusBadRequest)
		var body map[string]any
		readJSON(t, resp, &body)
		code := jsonAs[string](t, body["code"])
		if code != "bad_request" {
			t.Errorf("code = %q, want %q (must be snake_case, not Title Case)", code, "bad_request")
		}
	})

	t.Run("bearer auth takes precedence over session cookie", func(t *testing.T) {
		sessionToken := createTestUser(t, env, testEmail(t, "cookie-user"), "Cookie User", "password")
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
	})
}

func TestWebAuthConfig(t *testing.T) {
	t.Parallel()
	env := setupTestServer(t)

	newDisabledServer := func(t *testing.T) *httptest.Server {
		t.Helper()
		log := slog.New(slog.DiscardHandler)
		handler := &api.Handler{Pool: env.pool, Log: log, PasswordAuthEnabled: false}
		h, err := api.NewServer(handler, log)
		if err != nil {
			t.Fatalf("new server: %v", err)
		}
		srv := httptest.NewServer(api.MountWebAuth(h, handler))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("login disabled password auth", func(t *testing.T) {
		srv := newDisabledServer(t)
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/app/auth/login",
			strings.NewReader(`{"email":"test@example.com","password":"password"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		assertStatusClose(t, resp, http.StatusForbidden)
	})

	t.Run("auth config password enabled", func(t *testing.T) {
		resp := doRequestAs(t, env, "", "GET", "/app/auth/config", "")
		assertStatus(t, resp, http.StatusOK)
		var config map[string]any
		readJSON(t, resp, &config)
		password := jsonAs[map[string]any](t, config["password"])
		if password["enabled"] != true {
			t.Errorf("password.enabled = %v, want true", password["enabled"])
		}
	})

	t.Run("auth config password disabled", func(t *testing.T) {
		srv := newDisabledServer(t)
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/app/auth/config", nil)
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
	})
}
