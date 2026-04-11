package runtime

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var DefaultOAuthScopes = []string{
	"activity:read",
	"admin:read",
	"admin:write",
	"profile:read",
	"spaces:read",
	"spaces:write",
	"tags:read",
	"tags:write",
	"tasks:read",
	"tasks:write",
	"users:read",
	"users:write",
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func OAuthScopeString() string {
	return strings.Join(DefaultOAuthScopes, " ")
}

func GenerateOAuthState() (string, error) {
	return randomHex(16)
}

func GenerateCodeVerifier() (string, error) {
	return randomHex(32)
}

func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func ExchangeAuthorizationCode(ctx context.Context, server *url.URL, redirectURI string, code string, verifier string) (OAuthCredentials, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {"tend-cli"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}
	resp, err := postOAuthForm(ctx, resolveURL(server, "/oauth/token"), form)
	if err != nil {
		return OAuthCredentials{}, err
	}
	return oauthCredentialsFromResponse(resp), nil
}

func RefreshOAuthCredentials(ctx context.Context, server *url.URL, creds OAuthCredentials) (OAuthCredentials, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {creds.ClientID},
		"refresh_token": {creds.RefreshToken},
	}
	resp, err := postOAuthForm(ctx, resolveURL(server, "/oauth/token"), form)
	if err != nil {
		return OAuthCredentials{}, err
	}
	updated := oauthCredentialsFromResponse(resp)
	updated.ClientID = creds.ClientID
	return updated, nil
}

func RevokeOAuthToken(ctx context.Context, server *url.URL, token string) error {
	form := url.Values{"token": {token}}
	_, err := postOAuthForm(ctx, resolveURL(server, "/oauth/revoke"), form)
	return err
}

func OpenBrowser(rawURL string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{rawURL}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		cmd = "xdg-open"
		args = []string{rawURL}
	}
	return exec.Command(cmd, args...).Run()
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func postOAuthForm(ctx context.Context, endpoint *url.URL, form url.Values) (oauthTokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return oauthTokenResponse{}, fmt.Errorf("build oauth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return oauthTokenResponse{}, fmt.Errorf("request oauth endpoint: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return oauthTokenResponse{}, fmt.Errorf("read oauth error response: %w", err)
		}

		var oauthErr oauthErrorResponse
		if len(body) > 0 && json.Unmarshal(body, &oauthErr) == nil && (oauthErr.Error != "" || oauthErr.ErrorDescription != "") {
			return oauthTokenResponse{}, &APIError{
				StatusCode: resp.StatusCode,
				Code:       oauthErr.Error,
				Message:    oauthErr.ErrorDescription,
			}
		}

		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return oauthTokenResponse{}, &APIError{
			StatusCode: resp.StatusCode,
			Message:    message,
		}
	}

	var body oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return oauthTokenResponse{}, fmt.Errorf("decode oauth response: %w", err)
	}
	return body, nil
}

func oauthCredentialsFromResponse(resp oauthTokenResponse) OAuthCredentials {
	expiresAt := time.Now()
	if resp.ExpiresIn > 0 {
		expiresAt = expiresAt.Add(time.Duration(resp.ExpiresIn) * time.Second)
	}
	return OAuthCredentials{
		ClientID:     "tend-cli",
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		Scope:        resp.Scope,
		Expiry:       expiresAt,
	}
}
