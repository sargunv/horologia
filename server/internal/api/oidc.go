package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	zhttp "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// OIDCConfig holds the configuration for the OIDC relying party.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewOIDCHandler creates an http.Handler that handles the OIDC authorization
// code flow. It registers two routes:
//
//	GET /auth/oidc          → redirects to the OIDC provider
//	GET /auth/oidc/callback → handles the callback, creates a user+token
//
// Returns nil if OIDC is not configured.
func NewOIDCHandler(ctx context.Context, cfg OIDCConfig, handler *Handler) (http.Handler, error) {
	// Generate separate keys for HMAC authentication and AES encryption of OIDC state cookies.
	hashKey := make([]byte, 32)
	if _, err := rand.Read(hashKey); err != nil {
		return nil, fmt.Errorf("generate oidc hash key: %w", err)
	}
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("generate oidc encryption key: %w", err)
	}
	cookieHandler := zhttp.NewCookieHandler(hashKey, encKey)

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
	}

	provider, err := rp.NewRelyingPartyOIDC(ctx,
		cfg.Issuer,
		cfg.ClientID,
		cfg.ClientSecret,
		cfg.RedirectURL,
		[]string{"openid", "email", "profile"},
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("create oidc relying party: %w", err)
	}

	mux := http.NewServeMux()

	// GET /auth/oidc → redirect to IdP.
	mux.Handle("GET /auth/oidc", rp.AuthURLHandler(func() string {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			panic("generate oidc state: " + err.Error())
		}
		return hex.EncodeToString(b)
	}, provider))

	// GET /auth/oidc/callback → exchange code, find/create user, issue token.
	marshalToken := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, relyingParty rp.RelyingParty) {
		handleOIDCCallback(w, r, tokens, relyingParty, handler)
	}
	mux.Handle("GET /auth/oidc/callback", rp.CodeExchangeHandler(marshalToken, provider))

	return mux, nil
}

func handleOIDCCallback(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], relyingParty rp.RelyingParty, handler *Handler) {
	ctx := r.Context()

	subject := tokens.IDTokenClaims.Subject

	// The ID token may not contain email/profile claims (per OIDC spec,
	// they go in userinfo when an access token is issued). Fetch from userinfo.
	info, err := rp.Userinfo[*oidc.UserInfo](ctx, tokens.AccessToken, tokens.TokenType, subject, relyingParty)
	if err != nil {
		handler.Log.ErrorContext(ctx, "oidc: userinfo request failed", "error", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	email := info.Email
	if email == "" {
		email = tokens.IDTokenClaims.Email
	}
	if email == "" {
		handler.Log.ErrorContext(ctx, "oidc: no email from provider", "subject", subject)
		http.Error(w, "OIDC provider did not return an email address", http.StatusBadRequest)
		return
	}
	name := info.Name
	if name == "" {
		name = tokens.IDTokenClaims.Name
	}
	if name == "" {
		name = email
	}

	q := dbgen.New(handler.DB)
	subjectStr := subject

	// Try to find existing user by OIDC subject.
	user, err := q.GetUserByOIDCSubject(ctx, &subjectStr)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		handler.Log.ErrorContext(ctx, "oidc: get user", "error", err)
		http.Error(w, "failed to look up user", http.StatusInternalServerError)
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		// No user with this OIDC subject. Try matching by email to link
		// an existing password-created account.
		user, err = q.GetUserByEmail(ctx, email)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			handler.Log.ErrorContext(ctx, "oidc: get user by email", "error", err)
			http.Error(w, "failed to look up user", http.StatusInternalServerError)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			// Completely new user — create one.
			ts := now()
			user, err = q.CreateUser(ctx, dbgen.CreateUserParams{
				Email:       email,
				Name:        name,
				OidcSubject: &subjectStr,
				CreatedAt:   ts,
				UpdatedAt:   ts,
			})
			if err != nil {
				handler.Log.ErrorContext(ctx, "oidc: create user", "error", err)
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
			handler.Log.InfoContext(ctx, "oidc: created user", "email", email, "subject", subject)
		} else {
			// Existing user found by email — link the OIDC subject.
			if err := q.SetUserOIDCSubject(ctx, dbgen.SetUserOIDCSubjectParams{
				OidcSubject: &subjectStr,
				ID:          user.ID,
			}); err != nil {
				handler.Log.ErrorContext(ctx, "oidc: link user", "error", err)
				http.Error(w, "failed to link user", http.StatusInternalServerError)
				return
			}
			handler.Log.InfoContext(ctx, "oidc: linked user", "email", email, "subject", subject)
		}
	}

	// Generate a session token.
	raw, hash, err := generateToken()
	if err != nil {
		handler.Log.ErrorContext(ctx, "oidc: generate token", "error", err)
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: hash,
		Name:      "",
		Kind:      "session",
		CreatedAt: now(),
	})
	if err != nil {
		handler.Log.ErrorContext(ctx, "oidc: create token", "error", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	setSessionCookie(w, raw)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// MountOIDC wraps an ogen handler with optional OIDC routes.
// If oidcHandler is nil, returns the base handler unchanged.
func MountOIDC(base http.Handler, oidcHandler http.Handler, log *slog.Logger) http.Handler {
	if oidcHandler == nil {
		return base
	}
	mux := http.NewServeMux()
	mux.Handle("/auth/oidc", oidcHandler)
	mux.Handle("/auth/oidc/callback", oidcHandler)
	mux.Handle("/", base)
	log.Info("OIDC enabled")
	return mux
}
