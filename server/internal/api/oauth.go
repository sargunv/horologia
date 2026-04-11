package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

const (
	oauthCodeTTL         = 5 * time.Minute
	oauthAccessTokenTTL  = time.Hour
	oauthRefreshTokenTTL = 30 * 24 * time.Hour
)

var oauthAuthorizeTemplate = template.Must(template.New("oauth-authorize").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Authorize {{ .ClientDisplayName }}</title>
  <style>
    body { font-family: sans-serif; background: #f6f7f9; color: #111827; margin: 0; }
    .shell { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px; }
    .card { width: 100%; max-width: 520px; background: white; border: 1px solid #e5e7eb; border-radius: 16px; padding: 24px; box-shadow: 0 10px 30px rgba(0,0,0,0.06); }
    h1 { margin: 0 0 8px; font-size: 1.5rem; }
    p { color: #4b5563; line-height: 1.5; }
    ul { padding-left: 20px; }
    .scope { margin: 6px 0; }
    .actions { display: flex; gap: 12px; margin-top: 24px; }
    button { border-radius: 10px; padding: 10px 16px; font-size: 0.95rem; cursor: pointer; }
    .approve { background: #111827; color: white; border: none; }
    .deny { background: white; color: #111827; border: 1px solid #d1d5db; }
    .meta { margin-top: 16px; font-size: 0.9rem; color: #6b7280; }
  </style>
</head>
<body>
  <div class="shell">
    <div class="card">
      <h1>Authorize {{ .ClientDisplayName }}</h1>
      <p><strong>{{ .ClientDisplayName }}</strong> wants access to your Tend account for <strong>{{ .UserEmail }}</strong>.</p>
      <p>The client requested these permissions:</p>
      <ul>
      {{ range .Scopes }}
        <li class="scope"><code>{{ . }}</code></li>
      {{ end }}
      </ul>
      {{ if .Resource }}
      <p class="meta">Requested resource: <code>{{ .Resource }}</code></p>
      {{ end }}
      <form method="post" action="/oauth/authorize">
        <input type="hidden" name="response_type" value="{{ .ResponseType }}">
        <input type="hidden" name="client_id" value="{{ .ClientID }}">
        <input type="hidden" name="redirect_uri" value="{{ .RedirectURI }}">
        <input type="hidden" name="scope" value="{{ .Scope }}">
        <input type="hidden" name="state" value="{{ .State }}">
        <input type="hidden" name="code_challenge" value="{{ .CodeChallenge }}">
        <input type="hidden" name="code_challenge_method" value="{{ .CodeChallengeMethod }}">
        <input type="hidden" name="resource" value="{{ .Resource }}">
        <div class="actions">
          <button class="approve" type="submit" name="decision" value="approve">Authorize</button>
          <button class="deny" type="submit" name="decision" value="deny">Cancel</button>
        </div>
      </form>
    </div>
  </div>
</body>
</html>`))

type oauthAuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
}

// MountOAuth wraps the base handler with OAuth and protected-resource metadata routes.
func MountOAuth(base http.Handler, handler *Handler) http.Handler {
	mux := http.NewServeMux()
	oauthHandler := NewOAuthHandler(handler)
	mux.Handle("/.well-known/oauth-authorization-server", oauthHandler)
	mux.Handle("/.well-known/oauth-protected-resource", oauthHandler)
	mux.Handle("/api/.well-known/oauth-protected-resource", oauthHandler)
	mux.Handle("/mcp/.well-known/oauth-protected-resource", oauthHandler)
	mux.Handle("/oauth/authorize", oauthHandler)
	mux.Handle("/oauth/token", oauthHandler)
	mux.Handle("/oauth/revoke", oauthHandler)
	mux.Handle("/", base)
	return mux
}

// NewOAuthHandler creates handlers for Tend's OAuth metadata and protocol endpoints.
func NewOAuthHandler(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", handler.oauthAuthorizationServerMetadata)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", handler.oauthProtectedResourceMetadata)
	mux.HandleFunc("GET /api/.well-known/oauth-protected-resource", handler.oauthProtectedResourceMetadata)
	mux.HandleFunc("GET /mcp/.well-known/oauth-protected-resource", handler.oauthProtectedResourceMetadata)
	mux.HandleFunc("GET /oauth/authorize", handler.oauthAuthorize)
	mux.HandleFunc("POST /oauth/authorize", handler.oauthAuthorize)
	mux.HandleFunc("POST /oauth/token", handler.oauthToken)
	mux.HandleFunc("POST /oauth/revoke", handler.oauthRevoke)
	return mux
}

func (h *Handler) oauthAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	base := h.publicBaseURL(r)
	if base == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "public URL is not configured",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"scopes_supported":                      auth.SupportedScopes(),
	})
}

func (h *Handler) oauthProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	base := h.publicBaseURL(r)
	if base == "" {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "public URL is not configured",
		})
		return
	}
	resource := base + "/api"
	switch r.URL.Path {
	case "/mcp/.well-known/oauth-protected-resource":
		resource = base + "/mcp"
	case "/.well-known/oauth-protected-resource":
		resource = base + "/api"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 resource,
		"authorization_servers":    []string{base},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":         auth.SupportedScopes(),
	})
}

func (h *Handler) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.oauthAuthorizeGet(w, r)
		return
	}
	h.oauthAuthorizePost(w, r)
}

func (h *Handler) oauthAuthorizeGet(w http.ResponseWriter, r *http.Request) {
	req, client, scopes, err := h.validateAuthorizeRequest(r)
	if err != nil {
		h.oauthAuthorizeError(w, r, req, client, "invalid_request", err.Error())
		return
	}

	user, ok := h.oauthSessionUser(w, r)
	if !ok {
		return
	}

	if h.oauthHasConsent(r.Context(), user.ID, client.ClientID, scopes) {
		h.oauthIssueCodeRedirect(w, r, req, client, user, scopes)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = oauthAuthorizeTemplate.Execute(w, map[string]any{
		"ClientID":            client.ClientID,
		"ClientDisplayName":   client.DisplayName,
		"ResponseType":        req.ResponseType,
		"RedirectURI":         req.RedirectURI,
		"Scope":               strings.Join(scopes, " "),
		"Scopes":              scopes,
		"State":               req.State,
		"CodeChallenge":       req.CodeChallenge,
		"CodeChallengeMethod": req.CodeChallengeMethod,
		"Resource":            req.Resource,
		"UserEmail":           user.Email,
	})
}

func (h *Handler) oauthAuthorizePost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid form body")
		return
	}

	req, client, scopes, err := h.validateAuthorizeValues(r, r.PostForm)
	if err != nil {
		h.oauthAuthorizeError(w, r, req, client, "invalid_request", err.Error())
		return
	}

	user, ok := h.oauthSessionUser(w, r)
	if !ok {
		return
	}

	if r.PostForm.Get("decision") != "approve" {
		h.oauthAuthorizeError(w, r, req, client, "access_denied", "authorization denied")
		return
	}

	h.oauthIssueCodeRedirect(w, r, req, client, user, scopes)
}

func (h *Handler) oauthToken(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_request", "invalid form body")
		return
	}

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		h.oauthTokenAuthorizationCode(w, r)
	case "refresh_token":
		h.oauthTokenRefresh(w, r)
	default:
		h.writeOAuthJSONError(w, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type")
	}
}

func (h *Handler) oauthTokenAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimSpace(r.PostForm.Get("client_id"))
	code := strings.TrimSpace(r.PostForm.Get("code"))
	redirectURI := strings.TrimSpace(r.PostForm.Get("redirect_uri"))
	codeVerifier := strings.TrimSpace(r.PostForm.Get("code_verifier"))
	if clientID == "" || code == "" || redirectURI == "" || codeVerifier == "" {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_request", "client_id, code, redirect_uri, and code_verifier are required")
		return
	}

	q := dbgen.New(h.Pool)
	client, err := q.GetOAuthClient(r.Context(), clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_client", "unknown client_id")
		return
	}
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: get client", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	codeHash := auth.HashToken(code)
	authCode, err := q.GetOAuthAuthorizationCodeByHash(r.Context(), codeHash)
	if errors.Is(err, pgx.ErrNoRows) {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid")
		return
	}
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: get authorization code", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	now := time.Now()
	switch {
	case now.After(authCode.ExpiresAt.Time):
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code has expired")
		return
	case authCode.ClientID != client.ClientID:
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code client mismatch")
		return
	case authCode.RedirectUri != redirectURI:
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "redirect URI mismatch")
		return
	case authCode.CodeChallengeMethod != "S256":
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "unsupported code challenge method")
		return
	case oauthCodeChallengeS256(codeVerifier) != authCode.CodeChallenge:
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "code verifier is invalid")
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	tq := dbgen.New(tx)

	result, err := tq.DeleteOAuthAuthorizationCodeByHash(r.Context(), codeHash)
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: delete authorization code", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}
	if result.RowsAffected() == 0 {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid")
		return
	}

	accessToken, refreshToken, expiresIn, err := h.createOAuthTokenPair(r.Context(), tq, oauthTokenPairInput{
		UserID:      authCode.UserID,
		ClientID:    client.ClientID,
		DisplayName: client.DisplayName,
		Scopes:      append([]string(nil), authCode.Scopes...),
		Resource:    authCode.Resource.String,
		Now:         now,
	})
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: create token pair", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: commit token exchange", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	h.writeTokenResponse(w, accessToken, refreshToken, expiresIn, authCode.Scopes)
}

func (h *Handler) oauthTokenRefresh(w http.ResponseWriter, r *http.Request) {
	clientID := strings.TrimSpace(r.PostForm.Get("client_id"))
	refreshToken := strings.TrimSpace(r.PostForm.Get("refresh_token"))
	if clientID == "" || refreshToken == "" {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_request", "client_id and refresh_token are required")
		return
	}

	now := time.Now()
	user, err := auth.AuthenticateRefreshToken(r.Context(), h.Pool, refreshToken, now)
	if errors.Is(err, auth.ErrUnauthorized) {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid")
		return
	}
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: authenticate refresh token", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	if user.Token.ClientID != clientID {
		h.writeOAuthJSONError(w, http.StatusBadRequest, "invalid_grant", "refresh token client mismatch")
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	tq := dbgen.New(tx)

	if _, err := tq.DeleteAuthTokenByHash(r.Context(), auth.HashToken(refreshToken)); err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: delete refresh token", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	accessToken, newRefreshToken, expiresIn, err := h.createOAuthTokenPair(r.Context(), tq, oauthTokenPairInput{
		UserID:      user.ID,
		ClientID:    user.Token.ClientID,
		DisplayName: user.Token.Name,
		Scopes:      append([]string(nil), user.Token.Scopes...),
		Resource:    user.Token.Resource,
		Now:         now,
	})
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: refresh token pair", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: commit refresh exchange", "error", err)
		h.writeOAuthJSONError(w, http.StatusInternalServerError, "server_error", "internal server error")
		return
	}

	h.writeTokenResponse(w, accessToken, newRefreshToken, expiresIn, user.Token.Scopes)
}

func (h *Handler) oauthRevoke(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	token := strings.TrimSpace(r.PostForm.Get("token"))
	if token == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	q := dbgen.New(h.Pool)
	_, _ = q.DeleteAuthTokenByHash(r.Context(), auth.HashToken(token))
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) validateAuthorizeRequest(r *http.Request) (oauthAuthorizeRequest, dbgen.OauthClient, []string, error) {
	return h.validateAuthorizeValues(r, r.URL.Query())
}

func (h *Handler) validateAuthorizeValues(r *http.Request, values url.Values) (oauthAuthorizeRequest, dbgen.OauthClient, []string, error) {
	req := oauthAuthorizeRequest{
		ResponseType:        strings.TrimSpace(values.Get("response_type")),
		ClientID:            strings.TrimSpace(values.Get("client_id")),
		RedirectURI:         strings.TrimSpace(values.Get("redirect_uri")),
		Scope:               strings.TrimSpace(values.Get("scope")),
		State:               strings.TrimSpace(values.Get("state")),
		CodeChallenge:       strings.TrimSpace(values.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(values.Get("code_challenge_method")),
		Resource:            strings.TrimSpace(values.Get("resource")),
	}
	if req.ResponseType != "code" {
		return req, dbgen.OauthClient{}, nil, errors.New("response_type must be code")
	}
	if req.ClientID == "" || req.RedirectURI == "" {
		return req, dbgen.OauthClient{}, nil, errors.New("client_id and redirect_uri are required")
	}
	if req.CodeChallenge == "" {
		return req, dbgen.OauthClient{}, nil, errors.New("code_challenge is required")
	}
	if req.CodeChallengeMethod != "S256" {
		return req, dbgen.OauthClient{}, nil, errors.New("code_challenge_method must be S256")
	}

	scopes, err := auth.NormalizeScopes(req.Scope)
	if err != nil {
		return req, dbgen.OauthClient{}, nil, err
	}

	q := dbgen.New(h.Pool)
	client, err := q.GetOAuthClient(r.Context(), req.ClientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return req, dbgen.OauthClient{}, nil, errors.New("unknown client_id")
		}
		return req, dbgen.OauthClient{}, nil, err
	}

	if !oauthRedirectURIAllowed(client, req.RedirectURI) {
		return req, dbgen.OauthClient{}, nil, errors.New("redirect_uri is not registered for this client")
	}

	base := h.publicBaseURL(r)
	if req.Resource != "" && base == "" {
		return req, dbgen.OauthClient{}, nil, errors.New("public URL is not configured")
	}
	if req.Resource != "" && !oauthResourceAllowed(base, req.Resource) {
		return req, dbgen.OauthClient{}, nil, errors.New("resource is not supported")
	}

	return req, client, scopes, nil
}

func (h *Handler) oauthSessionUser(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		redirectTo := r.URL.Path
		if r.URL.RawQuery != "" {
			redirectTo += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(redirectTo), http.StatusSeeOther)
		return nil, false
	}

	user, err := auth.AuthenticateBearerToken(r.Context(), h.Pool, cookie.Value, time.Now())
	if errors.Is(err, auth.ErrUnauthorized) {
		h.clearSessionCookie(w)
		redirectTo := r.URL.Path
		if r.URL.RawQuery != "" {
			redirectTo += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(redirectTo), http.StatusSeeOther)
		return nil, false
	}
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: authenticate session", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return nil, false
	}

	return user, true
}

func (h *Handler) oauthHasConsent(ctx context.Context, userID int64, clientID string, scopes []string) bool {
	scopeKey := auth.ScopeSetKey(scopes)
	_, err := dbgen.New(h.Pool).GetOAuthConsentGrant(ctx, dbgen.GetOAuthConsentGrantParams{
		UserID:   userID,
		ClientID: clientID,
		ScopeKey: scopeKey,
	})
	return err == nil
}

func (h *Handler) oauthIssueCodeRedirect(w http.ResponseWriter, r *http.Request, req oauthAuthorizeRequest, client dbgen.OauthClient, user *auth.User, scopes []string) {
	now := time.Now()
	rawCode, codeHash, err := generateToken()
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: generate authorization code", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	q := dbgen.New(tx)

	_, err = q.CreateOAuthAuthorizationCode(r.Context(), dbgen.CreateOAuthAuthorizationCodeParams{
		CodeHash:            codeHash,
		UserID:              user.ID,
		ClientID:            client.ClientID,
		RedirectUri:         req.RedirectURI,
		Scopes:              scopes,
		Resource:            pgtype.Text{String: req.Resource, Valid: req.Resource != ""},
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           types.Timestamptz(now.Add(oauthCodeTTL)),
		CreatedAt:           types.Timestamptz(now),
	})
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: persist authorization code", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	_, err = q.UpsertOAuthConsentGrant(r.Context(), dbgen.UpsertOAuthConsentGrantParams{
		UserID:    user.ID,
		ClientID:  client.ClientID,
		ScopeKey:  auth.ScopeSetKey(scopes),
		Scopes:    scopes,
		CreatedAt: types.Timestamptz(now),
		UpdatedAt: types.Timestamptz(now),
	})
	if err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: persist consent grant", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		h.Log.ErrorContext(r.Context(), "oauth: commit authorization code", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	redirectTo := withQuery(req.RedirectURI, map[string]string{
		"code":  rawCode,
		"state": req.State,
	})
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

type oauthTokenPairInput struct {
	UserID      int64
	ClientID    string
	DisplayName string
	Scopes      []string
	Resource    string
	Now         time.Time
}

func (h *Handler) createOAuthTokenPair(ctx context.Context, q *dbgen.Queries, input oauthTokenPairInput) (accessToken string, refreshToken string, expiresIn int, err error) {
	accessToken, accessHash, err := generateToken()
	if err != nil {
		return "", "", 0, err
	}
	refreshToken, refreshHash, err := generateToken()
	if err != nil {
		return "", "", 0, err
	}

	displayName := input.DisplayName
	if strings.TrimSpace(displayName) == "" {
		displayName = input.ClientID
	}

	resource := pgtype.Text{String: input.Resource, Valid: input.Resource != ""}
	clientID := pgtype.Text{String: input.ClientID, Valid: input.ClientID != ""}

	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:        input.UserID,
		TokenHash:     accessHash,
		Name:          displayName,
		Kind:          dbgen.AuthTokenKindOauthAccess,
		ExpiresAt:     types.Timestamptz(input.Now.Add(oauthAccessTokenTTL)),
		CreatedAt:     types.Timestamptz(input.Now),
		OauthClientID: clientID,
		OauthScopes:   input.Scopes,
		OauthResource: resource,
	})
	if err != nil {
		return "", "", 0, err
	}

	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:        input.UserID,
		TokenHash:     refreshHash,
		Name:          displayName,
		Kind:          dbgen.AuthTokenKindOauthRefresh,
		ExpiresAt:     types.Timestamptz(input.Now.Add(oauthRefreshTokenTTL)),
		CreatedAt:     types.Timestamptz(input.Now),
		OauthClientID: clientID,
		OauthScopes:   input.Scopes,
		OauthResource: resource,
	})
	if err != nil {
		return "", "", 0, err
	}

	return accessToken, refreshToken, int(oauthAccessTokenTTL / time.Second), nil
}

func (h *Handler) oauthAuthorizeError(w http.ResponseWriter, r *http.Request, req oauthAuthorizeRequest, client dbgen.OauthClient, code string, description string) {
	if req.RedirectURI != "" && client.ClientID != "" && oauthRedirectURIAllowed(client, req.RedirectURI) {
		redirectTo := withQuery(req.RedirectURI, map[string]string{
			"error":             code,
			"error_description": description,
			"state":             req.State,
		})
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}
	h.writeOAuthJSONError(w, http.StatusBadRequest, code, description)
}

func (h *Handler) writeOAuthJSONError(w http.ResponseWriter, status int, code string, description string) {
	writeJSON(w, status, map[string]string{
		"error":             code,
		"error_description": description,
	})
}

func (h *Handler) writeTokenResponse(w http.ResponseWriter, accessToken string, refreshToken string, expiresIn int, scopes []string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": refreshToken,
		"scope":         strings.Join(scopes, " "),
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("oauth: writeJSON: failed to write response", "error", err)
	}
}

func oauthResourceAllowed(base string, resource string) bool {
	return resource == base+"/api" || resource == base+"/mcp"
}

func oauthRedirectURIAllowed(client dbgen.OauthClient, redirectURI string) bool {
	if slices.Contains(client.RedirectUris, redirectURI) {
		return true
	}
	if !client.LoopbackRedirects {
		return false
	}

	u, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}
	if u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func oauthCodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func withQuery(rawURL string, values map[string]string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	for key, value := range values {
		if value == "" {
			continue
		}
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (h *Handler) publicBaseURL(r *http.Request) string {
	if h.PublicURL != "" {
		return h.PublicURL
	}
	return auth.RequestPublicBaseURL(r)
}
