package api_test

import (
	"encoding/json"
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
	"github.com/oauth2-proxy/mockoidc"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// testOIDCUser wraps mockoidc.MockUser to include the "sub" claim in
// the userinfo response. mockoidc omits it, but the OIDC spec requires
// it and the zitadel RP library validates it.
// Upstream: https://github.com/oauth2-proxy/mockoidc/pull/45
type testOIDCUser struct {
	mockoidc.MockUser
}

func (u *testOIDCUser) Userinfo(scope []string) ([]byte, error) {
	base, err := u.MockUser.Userinfo(scope)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}
	m["sub"] = u.Subject
	return json.Marshal(m)
}

// queueOIDCUser queues a mock OIDC user for the next authorization request.
// It wraps mockoidc.MockUser in testOIDCUser to ensure the "sub" claim is
// included in the userinfo response.
func (e *testEnv) queueOIDCUser(subject, email string, emailVerified bool) {
	e.mock.QueueUser(&testOIDCUser{MockUser: mockoidc.MockUser{
		Subject:       subject,
		Email:         email,
		EmailVerified: emailVerified,
	}})
}

// newOIDCClient returns an http.Client with a cookie jar for driving the
// OIDC redirect chain. Each test should use its own client to avoid
// cookie cross-contamination.
func newOIDCClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("new cookie jar: %v", err)
	}
	return &http.Client{Jar: jar}
}

// setupOIDCEnv starts a mockoidc server and returns a test environment
// with OIDC routes enabled.
func setupOIDCEnv(t *testing.T) *testEnv {
	t.Helper()
	mock, err := mockoidc.Run()
	if err != nil {
		t.Fatalf("start mockoidc: %v", err)
	}
	t.Cleanup(func() { _ = mock.Shutdown() })
	return setupTestServer(t, withOIDC(mock))
}

// driveOIDCFlow initiates an OIDC login by hitting GET /auth/oidc and
// following all redirects through mockoidc and back to the callback.
// Returns the final response after the full redirect chain.
func driveOIDCFlow(t *testing.T, env *testEnv, client *http.Client, redirectPath string) *http.Response {
	t.Helper()
	u := env.Server.URL + "/auth/oidc"
	if redirectPath != "" {
		u += "?redirect=" + url.QueryEscape(redirectPath)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("drive oidc flow: %v", err)
	}
	return resp
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

	env.queueOIDCUser("new-oidc-subject", "oidc-user@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "/after-login")
	_ = resp.Body.Close()

	// The flow should end with a redirect; the final response is whatever
	// the server returns for the redirect target. We just need the session cookie.
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

	env.queueOIDCUser("returning-subject", "returning@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "")
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

	env.queueOIDCUser("link-oidc-subject", "link@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "")
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
	env.queueOIDCUser("new-subject", "overwrite@example.com", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "")
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

	env.queueOIDCUser("unverified-subject", "unverified@example.com", false)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want %d; body: %s", resp.StatusCode, http.StatusBadRequest, body)
	}
}

func TestOIDCLoginRedirectPreserved(t *testing.T) {
	env := setupOIDCEnv(t)

	env.queueOIDCUser("redirect-subject", "redirect@example.com", true)

	// Use a client that stops on the final redirect so we can inspect the
	// 303 response from the callback before the client follows it.
	serverHost := env.Server.URL
	client := newOIDCClient(t)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Stop when redirected to a non-auth path on the test server.
		if req.URL.Path != "/auth/oidc" &&
			req.URL.Path != "/auth/oidc/callback" &&
			strings.HasPrefix(req.URL.String(), serverHost) {
			return http.ErrUseLastResponse
		}
		return nil
	}

	resp := driveOIDCFlow(t, env, client, "/spaces")
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
	env.queueOIDCUser("malicious-redirect-subject", "malicious-redirect@example.com", true)

	// Pass a malicious redirect that isValidRedirect should reject.
	// The callback should fall back to "/".
	serverHost := env.Server.URL
	client := newOIDCClient(t)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Path != "/auth/oidc" &&
			req.URL.Path != "/auth/oidc/callback" &&
			strings.HasPrefix(req.URL.String(), serverHost) {
			return http.ErrUseLastResponse
		}
		return nil
	}

	resp := driveOIDCFlow(t, env, client, "//evil.com")
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
	env.queueOIDCUser("no-email-subject", "", true)

	client := newOIDCClient(t)
	resp := driveOIDCFlow(t, env, client, "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("got status %d, want %d; body: %s", resp.StatusCode, http.StatusBadRequest, body)
	}
}
