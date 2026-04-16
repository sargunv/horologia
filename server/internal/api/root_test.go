package api_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sargunv/horologia/server/internal/api"
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
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
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
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
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

func TestMountRootExposesOAuthAuthorizeRoute(t *testing.T) {
	env := setupTestServer(t)

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/oauth/authorize?"+url.Values{
		"response_type":         {"code"},
		"client_id":             {"horologia-cli"},
		"redirect_uri":          {"http://127.0.0.1:8484/callback"},
		"scope":                 {"profile:read"},
		"state":                 {"test-state"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}.Encode(), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	assertStatus(t, resp, http.StatusSeeOther)

	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatal("missing redirect location")
	}
	if location[:16] != "/login?redirect=" {
		t.Fatalf("location = %q, want login redirect", location)
	}
	_ = resp.Body.Close()
}

func TestInternalAPIsRejectCrossOriginRequests(t *testing.T) {
	env := setupTestServer(t)

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/app/auth/config", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("auth config: %v", err)
	}
	assertStatus(t, resp, http.StatusForbidden)
}

func TestPublicAPIsAllowCrossOriginRequests(t *testing.T) {
	env := setupTestServer(t)

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
}

func TestAPICookieAuthRequiresSameOrigin(t *testing.T) {
	env := setupTestServer(t)
	sessionToken := createTestUser(t, env, "cookie-root@example.com", "Cookie Root", "password")

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/users/me", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: "horologia_session", Value: sessionToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("same-origin api: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	req, err = http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/users/me", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: "horologia_session", Value: sessionToken})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cross-origin api: %v", err)
	}
	assertStatusClose(t, resp, http.StatusForbidden)
}

func TestOAuthAuthorizePostRejectsCrossOriginRequests(t *testing.T) {
	env := setupTestServer(t)

	log := slog.New(slog.DiscardHandler)
	root := api.MountRoot(env.Server.Config.Handler, nil, env.pool, log, "")
	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/app/oauth/authorize", strings.NewReader("decision=approve"))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	req.AddCookie(&http.Cookie{Name: "horologia_session", Value: env.Token})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("oauth authorize: %v", err)
	}
	assertStatusClose(t, resp, http.StatusForbidden)
}
