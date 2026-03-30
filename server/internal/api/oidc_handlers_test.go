package api_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// newOIDCClient returns an http.Client with a cookie jar for OIDC tests.
// Each test should use its own client to avoid cookie cross-contamination.
func newOIDCClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("new cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

// setupOIDCEnv returns a test environment with a zitadel example OP
// and OIDC routes enabled.
func setupOIDCEnv(t *testing.T) *testEnv {
	t.Helper()
	return setupTestServer(t, withOIDC())
}

// driveOIDCFlow drives the full OIDC authorization code flow:
//  1. GET /auth/oidc → follows redirects to the OP's login form
//  2. POST credentials to the login form
//  3. Follows remaining redirects (OP callback → our callback → final redirect)
//
// The username parameter is the OIDC user's subject (used as username in the OP).
func driveOIDCFlow(t *testing.T, env *testEnv, client *http.Client, username, redirectPath string) *http.Response {
	t.Helper()

	ctx := t.Context()

	// Phase 1: initiate OIDC flow — follows redirects until the OP login form (200 OK).
	u := env.Server.URL + "/auth/oidc"
	if redirectPath != "" {
		u += "?redirect=" + url.QueryEscape(redirectPath)
	}
	initReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	loginResp, err := client.Do(initReq)
	if err != nil {
		t.Fatalf("initiate oidc flow: %v", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(loginResp.Body)
		_ = loginResp.Body.Close()
		t.Fatalf("initiate oidc flow: got status %d, want 200; body: %s; url: %s",
			loginResp.StatusCode, body, loginResp.Request.URL)
	}
	_ = loginResp.Body.Close()

	// The final URL after redirects is the OP's login form with ?authRequestID=...
	authRequestID := loginResp.Request.URL.Query().Get("authRequestID")
	if authRequestID == "" {
		t.Fatal("login page missing authRequestID parameter")
	}

	// Phase 2: POST credentials to the OP login form.
	// This follows redirects through OP auth callback → our /auth/oidc/callback → final redirect.
	loginURL := loginResp.Request.URL.String()
	form := url.Values{
		"username": {username},
		"password": {testOIDCUserPassword},
		"id":       {authRequestID},
	}
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("new login request: %v", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(postReq)
	if err != nil {
		t.Fatalf("submit oidc login: %v", err)
	}
	return resp
}

// stopAfterCallback configures the client to stop following redirects once
// the OIDC callback redirects to an app route, so the test can inspect the
// 303 response. It allows redirects to /auth/oidc, /auth/oidc/callback, and
// any non-server URL (the OP) to proceed normally.
func stopAfterCallback(client *http.Client, serverURL string) {
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Path != "/auth/oidc" &&
			req.URL.Path != "/auth/oidc/callback" &&
			strings.HasPrefix(req.URL.String(), serverURL) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

// extractSessionCookie returns the tend_session cookie value from the
// client's cookie jar for the given server URL. Returns "" if not present.
func extractSessionCookie(t *testing.T, client *http.Client, serverURL string) string {
	t.Helper()
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == "tend_session" {
			return c.Value
		}
	}
	return ""
}

func TestOIDCLoginNewUser(t *testing.T) {
	env := setupOIDCEnv(t)

	env.addOIDCUser("new-oidc-subject", "oidc-user@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "new-oidc-subject", "/after-login")
	_ = resp.Body.Close()

	sessionToken := extractSessionCookie(t, client, env.Server.URL)
	if sessionToken == "" {
		t.Fatal("expected tend_session cookie after OIDC login")
	}

	// Verify the session works.
	meResp := doRequestAs(t, env, sessionToken, "GET", "/users/me", "")
	assertStatus(t, meResp, http.StatusOK)
	var me map[string]any
	readJSON(t, meResp, &me)
	if me["email"] != "oidc-user@example.com" {
		t.Errorf("email = %v, want oidc-user@example.com", me["email"])
	}

	// Verify the DB has the OIDC subject set.
	q := dbgen.New(env.pool)
	user, err := q.GetUserByOIDCSubject(t.Context(), pgtype.Text{String: "new-oidc-subject", Valid: true})
	if err != nil {
		t.Fatalf("get user by oidc subject: %v", err)
	}
	if user.Email != "oidc-user@example.com" {
		t.Errorf("db email = %v, want oidc-user@example.com", user.Email)
	}
}

func TestOIDCLoginReturningUser(t *testing.T) {
	env := setupOIDCEnv(t)
	ctx := t.Context()

	// Pre-create a user with an OIDC subject directly in the DB.
	q := dbgen.New(env.pool)
	now := types.Timestamptz(time.Now())
	existingUser, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:       "returning@example.com",
		Name:        "Returning User",
		OidcSubject: pgtype.Text{String: "returning-subject", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	env.addOIDCUser("returning-subject", "returning@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "returning-subject", "")
	_ = resp.Body.Close()

	sessionToken := extractSessionCookie(t, client, env.Server.URL)
	if sessionToken == "" {
		t.Fatal("expected tend_session cookie after OIDC login")
	}

	// Verify it's the same user.
	meResp := doRequestAs(t, env, sessionToken, "GET", "/users/me", "")
	assertStatus(t, meResp, http.StatusOK)
	var me map[string]any
	readJSON(t, meResp, &me)
	if me["email"] != "returning@example.com" {
		t.Errorf("email = %v, want returning@example.com", me["email"])
	}
	meID, err := types.ParseUserID(jsonAs[string](t, me["id"]))
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	if meID != existingUser.ID {
		t.Errorf("user ID = %d, want %d (should be same user)", meID, existingUser.ID)
	}
}

func TestOIDCLoginEmailAutoLink(t *testing.T) {
	env := setupOIDCEnv(t)

	// Create a password-based user with no OIDC subject.
	existingUser, err := taskengine.CreateUserWithPassword(t.Context(), env.pool, "link@example.com", "Link User", "password123", false, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	env.addOIDCUser("link-oidc-subject", "link@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "link-oidc-subject", "")
	_ = resp.Body.Close()

	sessionToken := extractSessionCookie(t, client, env.Server.URL)
	if sessionToken == "" {
		t.Fatal("expected tend_session cookie after OIDC login")
	}

	// Verify the session works, returns the same email, and reuses the existing user.
	meResp := doRequestAs(t, env, sessionToken, "GET", "/users/me", "")
	assertStatus(t, meResp, http.StatusOK)
	var me map[string]any
	readJSON(t, meResp, &me)
	if me["email"] != "link@example.com" {
		t.Errorf("email = %v, want link@example.com", me["email"])
	}
	meID, err := types.ParseUserID(jsonAs[string](t, me["id"]))
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	if meID != existingUser.ID {
		t.Errorf("user ID = %d, want %d (should reuse existing user)", meID, existingUser.ID)
	}

	// Verify the OIDC subject was linked in the DB.
	q := dbgen.New(env.pool)
	user, err := q.GetUserByOIDCSubject(t.Context(), pgtype.Text{String: "link-oidc-subject", Valid: true})
	if err != nil {
		t.Fatalf("get user by oidc subject: %v", err)
	}
	if user.Email != "link@example.com" {
		t.Errorf("db email = %v, want link@example.com", user.Email)
	}
}

func TestOIDCLoginEmailAutoLinkUpdatesSubject(t *testing.T) {
	env := setupOIDCEnv(t)
	ctx := t.Context()

	// Create a user with an existing OIDC subject. When a new OIDC login
	// has a different subject but the same email, the subject is updated.
	q := dbgen.New(env.pool)
	now := types.Timestamptz(time.Now())
	_, err := q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:       "overwrite@example.com",
		Name:        "Overwrite User",
		OidcSubject: pgtype.Text{String: "old-subject", Valid: true},
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Login via OIDC with a different subject but the same email.
	env.addOIDCUser("new-subject", "overwrite@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "new-subject", "")
	_ = resp.Body.Close()

	sessionToken := extractSessionCookie(t, client, env.Server.URL)
	if sessionToken == "" {
		t.Fatal("expected tend_session cookie after OIDC login")
	}

	// Verify the OIDC subject was overwritten in the DB.
	user, err := q.GetUserByOIDCSubject(ctx, pgtype.Text{String: "new-subject", Valid: true})
	if err != nil {
		t.Fatalf("get user by new oidc subject: %v", err)
	}
	if user.Email != "overwrite@example.com" {
		t.Errorf("db email = %v, want overwrite@example.com", user.Email)
	}

	// Verify the old subject no longer resolves.
	_, err = q.GetUserByOIDCSubject(ctx, pgtype.Text{String: "old-subject", Valid: true})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("old OIDC subject still resolves after overwrite: %v", err)
	}
}

func TestOIDCLoginEmailUnverified(t *testing.T) {
	env := setupOIDCEnv(t)

	env.addOIDCUser("unverified-subject", "unverified@example.com", false)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "unverified-subject", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want %d; body: %s", resp.StatusCode, http.StatusBadRequest, body)
	}

	// No session cookie should be issued on rejection.
	if tok := extractSessionCookie(t, client, env.Server.URL); tok != "" {
		t.Error("session cookie should not be set when email is unverified")
	}
}

func TestOIDCLoginRedirectPreserved(t *testing.T) {
	env := setupOIDCEnv(t)

	env.addOIDCUser("redirect-subject", "redirect@example.com", true)

	// Use a client that stops on the final redirect so we can inspect the
	// 303 response from the callback before the client follows it.
	client := newOIDCClient(t)
	stopAfterCallback(client, env.Server.URL)

	resp := driveOIDCFlow(t, env, client, "redirect-subject", "/spaces")
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	if loc != "/spaces" {
		t.Errorf("redirect location = %q, want /spaces", loc)
	}
}

func TestOIDCLoginRedirectMaliciousIgnored(t *testing.T) {
	env := setupOIDCEnv(t)
	env.addOIDCUser("malicious-redirect-subject", "malicious-redirect@example.com", true)

	client := newOIDCClient(t)
	stopAfterCallback(client, env.Server.URL)

	resp := driveOIDCFlow(t, env, client, "malicious-redirect-subject", "//evil.com")
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusSeeOther)
	}
	loc := resp.Header.Get("Location")
	if loc != "/" {
		t.Errorf("redirect location = %q, want / (malicious redirect should be ignored)", loc)
	}
}

func TestOIDCLoginMissingEmail(t *testing.T) {
	env := setupOIDCEnv(t)

	// User with valid username but empty email — the OP authenticates
	// successfully but our handler rejects the empty email.
	env.addOIDCUser("no-email-subject", "", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "no-email-subject", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want %d; body: %s", resp.StatusCode, http.StatusBadRequest, body)
	}

	// No session cookie should be issued on rejection.
	if tok := extractSessionCookie(t, client, env.Server.URL); tok != "" {
		t.Error("session cookie should not be set when email is missing")
	}
}
