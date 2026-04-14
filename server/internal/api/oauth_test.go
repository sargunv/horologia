package api_test

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/sargunv/horologia/server/internal/auth"
)

func TestOAuthAuthorizationServerMetadata(t *testing.T) {
	env := setupTestServer(t)

	resp, err := doOAuthRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/.well-known/oauth-authorization-server", nil, "")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	readJSON(t, resp, &body)
	if body["issuer"] != env.Server.URL {
		t.Fatalf("issuer = %v, want %s", body["issuer"], env.Server.URL)
	}
	if body["authorization_endpoint"] != env.Server.URL+"/oauth/authorize" {
		t.Fatalf("authorization_endpoint = %v", body["authorization_endpoint"])
	}
}

func TestOAuthProtectedResourceMetadataForMCP(t *testing.T) {
	env := setupTestServer(t)

	resp, err := doOAuthRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/mcp/.well-known/oauth-protected-resource", nil, "")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var body map[string]any
	readJSON(t, resp, &body)
	if body["resource"] != env.Server.URL+"/mcp" {
		t.Fatalf("resource = %v, want %s", body["resource"], env.Server.URL+"/mcp")
	}
}

func TestOAuthAuthorizeRequiresLogin(t *testing.T) {
	env := setupTestServer(t)

	client := noRedirectClient(t)
	resp, err := doOAuthRequest(t, client, http.MethodGet, env.Server.URL+"/oauth/authorize?"+url.Values{
		"response_type":         {"code"},
		"client_id":             {"horologia-cli"},
		"redirect_uri":          {"http://127.0.0.1:8484/callback"},
		"scope":                 {"profile:read"},
		"state":                 {"test-state"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
	}.Encode(), nil, "")
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	assertStatus(t, resp, http.StatusSeeOther)

	location := resp.Header.Get("Location")
	if !strings.HasPrefix(location, "/login?redirect=") {
		t.Fatalf("Location = %q, want login redirect", location)
	}
}

func TestOAuthAuthorizationCodeFlow(t *testing.T) {
	env := setupTestServer(t)
	client := newOAuthBrowserClient(t)
	loginOAuthBrowserSession(t, env, client)

	verifier := "oauth-verifier-value-1234567890"
	challenge := oauthCodeChallengeS256(verifier)
	code := authorizeOAuthCode(t, env, client, url.Values{
		"response_type":         {"code"},
		"client_id":             {"horologia-cli"},
		"redirect_uri":          {"http://127.0.0.1:8484/callback"},
		"scope":                 {"profile:read spaces:read"},
		"state":                 {"opaque-state"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"resource":              {env.Server.URL + "/api"},
	})

	tokenResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"horologia-cli"},
		"code":          {code},
		"redirect_uri":  {"http://127.0.0.1:8484/callback"},
		"code_verifier": {verifier},
	}.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	assertStatus(t, tokenResp, http.StatusOK)

	var tokenBody map[string]any
	readJSON(t, tokenResp, &tokenBody)
	accessToken := jsonAs[string](t, tokenBody["access_token"])
	refreshToken := jsonAs[string](t, tokenBody["refresh_token"])
	if accessToken == "" || refreshToken == "" {
		t.Fatal("missing tokens in oauth response")
	}

	meResp := doRequestAs(t, env, accessToken, http.MethodGet, "/users/me", "")
	assertStatusClose(t, meResp, http.StatusOK)

	refreshAsBearerResp := doRequestAs(t, env, refreshToken, http.MethodGet, "/users/me", "")
	assertStatusClose(t, refreshAsBearerResp, http.StatusUnauthorized)

	refreshResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"horologia-cli"},
		"refresh_token": {refreshToken},
	}.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("refresh exchange: %v", err)
	}
	assertStatus(t, refreshResp, http.StatusOK)
	var refreshBody map[string]any
	readJSON(t, refreshResp, &refreshBody)
	newAccessToken := jsonAs[string](t, refreshBody["access_token"])
	if newAccessToken == accessToken {
		t.Fatal("refresh did not rotate the access token")
	}

	reuseResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {"horologia-cli"},
		"refresh_token": {refreshToken},
	}.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("reused refresh exchange: %v", err)
	}
	assertStatus(t, reuseResp, http.StatusBadRequest)
	var reuseBody map[string]any
	readJSON(t, reuseResp, &reuseBody)
	if reuseBody["error"] != "invalid_grant" {
		t.Fatalf("error = %v, want invalid_grant", reuseBody["error"])
	}

	revokeResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/revoke", strings.NewReader(url.Values{
		"token": {newAccessToken},
	}.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	assertStatusClose(t, revokeResp, http.StatusOK)

	assertStatusClose(t, doRequestAs(t, env, newAccessToken, http.MethodGet, "/users/me", ""), http.StatusUnauthorized)
}

func TestOAuthAuthorizationServerMetadataRejectsUntrustedDynamicHost(t *testing.T) {
	env := setupTestServer(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/.well-known/oauth-authorization-server", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "attacker.example"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	assertStatusClose(t, resp, http.StatusInternalServerError)
}

func TestOAuthAuthorizeAcceptsConfiguredPublicResourceWhenProxied(t *testing.T) {
	env := setupTestServer(t)
	env.Handler.PublicURL = env.Server.URL

	client := newOAuthBrowserClient(t)
	loginOAuthBrowserSession(t, env, client)

	form := url.Values{
		"response_type":         {"code"},
		"client_id":             {"horologia-cli"},
		"redirect_uri":          {"http://127.0.0.1:8484/callback"},
		"scope":                 {"profile:read"},
		"state":                 {"proxy-state"},
		"code_challenge":        {"challenge"},
		"code_challenge_method": {"S256"},
		"resource":              {env.Server.URL + "/api"},
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, env.Server.URL+"/oauth/authorize?"+form.Encode(), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Host = "localhost:5173"
	serverURL, err := url.Parse(env.Server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	for _, cookie := range client.Jar.Cookies(serverURL) {
		req.AddCookie(cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read consent page: %v", err)
	}
	_ = resp.Body.Close()
	if !strings.Contains(string(bodyBytes), "Authorize Horologia CLI") {
		t.Fatalf("consent page did not mention Tend CLI: %s", string(bodyBytes))
	}
}

func TestOAuthAuthorizeRejectsInvalidRequests(t *testing.T) {
	env := setupTestServer(t)

	testCases := []struct {
		name               string
		params             url.Values
		wantError          string
		wantErrorSubstring string
	}{
		{
			name: "unknown client",
			params: url.Values{
				"response_type":         {"code"},
				"client_id":             {"missing-client"},
				"redirect_uri":          {"http://127.0.0.1:8484/callback"},
				"scope":                 {"profile:read"},
				"state":                 {"test-state"},
				"code_challenge":        {"challenge"},
				"code_challenge_method": {"S256"},
			},
			wantError:          "invalid_request",
			wantErrorSubstring: "unknown client_id",
		},
		{
			name: "invalid redirect uri",
			params: url.Values{
				"response_type":         {"code"},
				"client_id":             {"horologia-cli"},
				"redirect_uri":          {"https://example.com/callback"},
				"scope":                 {"profile:read"},
				"state":                 {"test-state"},
				"code_challenge":        {"challenge"},
				"code_challenge_method": {"S256"},
			},
			wantError:          "invalid_request",
			wantErrorSubstring: "redirect_uri is not registered",
		},
		{
			name: "unsupported scope",
			params: url.Values{
				"response_type":         {"code"},
				"client_id":             {"horologia-cli"},
				"redirect_uri":          {"http://127.0.0.1:8484/callback"},
				"scope":                 {"profile:read nope:read"},
				"state":                 {"test-state"},
				"code_challenge":        {"challenge"},
				"code_challenge_method": {"S256"},
			},
			wantError:          "invalid_request",
			wantErrorSubstring: `unsupported scope "nope:read"`,
		},
		{
			name: "missing pkce challenge",
			params: url.Values{
				"response_type":         {"code"},
				"client_id":             {"horologia-cli"},
				"redirect_uri":          {"http://127.0.0.1:8484/callback"},
				"scope":                 {"profile:read"},
				"state":                 {"test-state"},
				"code_challenge_method": {"S256"},
			},
			wantError:          "invalid_request",
			wantErrorSubstring: "code_challenge is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := doOAuthRequest(t, http.DefaultClient, http.MethodGet, env.Server.URL+"/oauth/authorize?"+tc.params.Encode(), nil, "")
			if err != nil {
				t.Fatalf("authorize: %v", err)
			}
			assertStatus(t, resp, http.StatusBadRequest)

			var body map[string]any
			readJSON(t, resp, &body)
			if got := jsonAs[string](t, body["error"]); got != tc.wantError {
				t.Fatalf("error = %q, want %q", got, tc.wantError)
			}
			description := jsonAs[string](t, body["error_description"])
			if !strings.Contains(description, tc.wantErrorSubstring) {
				t.Fatalf("error_description = %q, want substring %q", description, tc.wantErrorSubstring)
			}
		})
	}
}

func TestOAuthTokenRejectsInvalidAuthorizationCodeExchanges(t *testing.T) {
	env := setupTestServer(t)

	t.Run("wrong verifier", func(t *testing.T) {
		client := newOAuthBrowserClient(t)
		loginOAuthBrowserSession(t, env, client)

		verifier := "oauth-verifier-value-1234567890"
		code := authorizeOAuthCode(t, env, client, url.Values{
			"response_type":         {"code"},
			"client_id":             {"horologia-cli"},
			"redirect_uri":          {"http://127.0.0.1:8484/callback"},
			"scope":                 {"profile:read"},
			"state":                 {"opaque-state"},
			"code_challenge":        {oauthCodeChallengeS256(verifier)},
			"code_challenge_method": {"S256"},
			"resource":              {env.Server.URL + "/api"},
		})

		resp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"horologia-cli"},
			"code":          {code},
			"redirect_uri":  {"http://127.0.0.1:8484/callback"},
			"code_verifier": {"wrong-verifier"},
		}.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatalf("token exchange: %v", err)
		}
		assertStatus(t, resp, http.StatusBadRequest)

		var body map[string]any
		readJSON(t, resp, &body)
		if got := jsonAs[string](t, body["error"]); got != "invalid_grant" {
			t.Fatalf("error = %q, want invalid_grant", got)
		}
		if got := jsonAs[string](t, body["error_description"]); got != "code verifier is invalid" {
			t.Fatalf("error_description = %q, want %q", got, "code verifier is invalid")
		}
	})

	t.Run("redirect uri mismatch", func(t *testing.T) {
		client := newOAuthBrowserClient(t)
		loginOAuthBrowserSession(t, env, client)

		verifier := "oauth-verifier-value-redirect"
		code := authorizeOAuthCode(t, env, client, url.Values{
			"response_type":         {"code"},
			"client_id":             {"horologia-cli"},
			"redirect_uri":          {"http://127.0.0.1:8484/callback"},
			"scope":                 {"profile:read"},
			"state":                 {"opaque-state"},
			"code_challenge":        {oauthCodeChallengeS256(verifier)},
			"code_challenge_method": {"S256"},
			"resource":              {env.Server.URL + "/api"},
		})

		resp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"horologia-cli"},
			"code":          {code},
			"redirect_uri":  {"http://127.0.0.1:9999/callback"},
			"code_verifier": {verifier},
		}.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatalf("token exchange: %v", err)
		}
		assertStatus(t, resp, http.StatusBadRequest)

		var body map[string]any
		readJSON(t, resp, &body)
		if got := jsonAs[string](t, body["error"]); got != "invalid_grant" {
			t.Fatalf("error = %q, want invalid_grant", got)
		}
		if got := jsonAs[string](t, body["error_description"]); got != "redirect URI mismatch" {
			t.Fatalf("error_description = %q, want %q", got, "redirect URI mismatch")
		}
	})

	t.Run("expired code", func(t *testing.T) {
		client := newOAuthBrowserClient(t)
		loginOAuthBrowserSession(t, env, client)

		verifier := "oauth-verifier-value-expired"
		code := authorizeOAuthCode(t, env, client, url.Values{
			"response_type":         {"code"},
			"client_id":             {"horologia-cli"},
			"redirect_uri":          {"http://127.0.0.1:8484/callback"},
			"scope":                 {"profile:read"},
			"state":                 {"opaque-state"},
			"code_challenge":        {oauthCodeChallengeS256(verifier)},
			"code_challenge_method": {"S256"},
			"resource":              {env.Server.URL + "/api"},
		})

		if _, err := env.pool.Exec(t.Context(), "UPDATE oauth_authorization_codes SET expires_at = now() - interval '1 second' WHERE code_hash = $1", auth.HashToken(code)); err != nil {
			t.Fatalf("expire authorization code: %v", err)
		}

		resp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"horologia-cli"},
			"code":          {code},
			"redirect_uri":  {"http://127.0.0.1:8484/callback"},
			"code_verifier": {verifier},
		}.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatalf("token exchange: %v", err)
		}
		assertStatus(t, resp, http.StatusBadRequest)

		var body map[string]any
		readJSON(t, resp, &body)
		if got := jsonAs[string](t, body["error"]); got != "invalid_grant" {
			t.Fatalf("error = %q, want invalid_grant", got)
		}
		if got := jsonAs[string](t, body["error_description"]); got != "authorization code has expired" {
			t.Fatalf("error_description = %q, want %q", got, "authorization code has expired")
		}
	})

	t.Run("reused code", func(t *testing.T) {
		client := newOAuthBrowserClient(t)
		loginOAuthBrowserSession(t, env, client)

		verifier := "oauth-verifier-value-reused"
		code := authorizeOAuthCode(t, env, client, url.Values{
			"response_type":         {"code"},
			"client_id":             {"horologia-cli"},
			"redirect_uri":          {"http://127.0.0.1:8484/callback"},
			"scope":                 {"profile:read"},
			"state":                 {"opaque-state"},
			"code_challenge":        {oauthCodeChallengeS256(verifier)},
			"code_challenge_method": {"S256"},
			"resource":              {env.Server.URL + "/api"},
		})

		firstResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"horologia-cli"},
			"code":          {code},
			"redirect_uri":  {"http://127.0.0.1:8484/callback"},
			"code_verifier": {verifier},
		}.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatalf("first token exchange: %v", err)
		}
		assertStatusClose(t, firstResp, http.StatusOK)

		reuseResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/token", strings.NewReader(url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"horologia-cli"},
			"code":          {code},
			"redirect_uri":  {"http://127.0.0.1:8484/callback"},
			"code_verifier": {verifier},
		}.Encode()), "application/x-www-form-urlencoded")
		if err != nil {
			t.Fatalf("reused token exchange: %v", err)
		}
		assertStatus(t, reuseResp, http.StatusBadRequest)

		var body map[string]any
		readJSON(t, reuseResp, &body)
		if got := jsonAs[string](t, body["error"]); got != "invalid_grant" {
			t.Fatalf("error = %q, want invalid_grant", got)
		}
		if got := jsonAs[string](t, body["error_description"]); got != "authorization code is invalid" {
			t.Fatalf("error_description = %q, want %q", got, "authorization code is invalid")
		}
	})
}

func noRedirectClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func oauthCodeChallengeS256(verifier string) string {
	sum := sha256Sum(verifier)
	return base64.RawURLEncoding.EncodeToString(sum)
}

func sha256Sum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func newOAuthBrowserClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func loginOAuthBrowserSession(t *testing.T, env *testEnv, client *http.Client) {
	t.Helper()
	loginReq, err := http.NewRequestWithContext(t.Context(), http.MethodPost, env.Server.URL+"/auth/login", strings.NewReader(`{"email":"test@example.com","password":"password"}`))
	if err != nil {
		t.Fatalf("new login request: %v", err)
	}
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp, err := client.Do(loginReq)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	assertStatusClose(t, loginResp, http.StatusOK)
}

func authorizeOAuthCode(t *testing.T, env *testEnv, client *http.Client, form url.Values) string {
	t.Helper()

	authResp, err := doOAuthRequest(t, client, http.MethodGet, env.Server.URL+"/oauth/authorize?"+form.Encode(), nil, "")
	if err != nil {
		t.Fatalf("authorize GET: %v", err)
	}
	if authResp.StatusCode == http.StatusFound {
		redirectLocation := authResp.Header.Get("Location")
		_ = authResp.Body.Close()
		return authorizationCodeFromRedirect(t, form, redirectLocation)
	}

	assertStatus(t, authResp, http.StatusOK)
	bodyBytes, err := io.ReadAll(authResp.Body)
	if err != nil {
		t.Fatalf("read consent page: %v", err)
	}
	_ = authResp.Body.Close()
	if !strings.Contains(string(bodyBytes), "Authorize Horologia CLI") {
		t.Fatalf("consent page did not mention Tend CLI: %s", string(bodyBytes))
	}

	postForm := url.Values{}
	for key, values := range form {
		postForm[key] = append([]string(nil), values...)
	}
	postForm.Set("decision", "approve")

	postResp, err := doOAuthRequest(t, client, http.MethodPost, env.Server.URL+"/oauth/authorize", strings.NewReader(postForm.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatalf("authorize POST: %v", err)
	}
	assertStatus(t, postResp, http.StatusFound)
	redirectLocation := postResp.Header.Get("Location")
	_ = postResp.Body.Close()

	return authorizationCodeFromRedirect(t, form, redirectLocation)
}

func authorizationCodeFromRedirect(t *testing.T, form url.Values, redirectLocation string) string {
	t.Helper()

	redirectURL, err := url.Parse(redirectLocation)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if state := form.Get("state"); state != "" && redirectURL.Query().Get("state") != state {
		t.Fatalf("state = %q, want %q", redirectURL.Query().Get("state"), state)
	}
	code := redirectURL.Query().Get("code")
	if code == "" {
		t.Fatal("missing authorization code")
	}
	return code
}

func doOAuthRequest(t *testing.T, client *http.Client, method string, rawURL string, body io.Reader, contentType string) (*http.Response, error) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return client.Do(req)
}
