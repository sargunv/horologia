package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	// Generate a random encryption key for OIDC state cookies.
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate oidc key: %w", err)
	}
	cookieHandler := zhttp.NewCookieHandler(key, key)

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
		rand.Read(b)
		return hex.EncodeToString(b)
	}, provider))

	// GET /auth/oidc/callback → exchange code, find/create user, issue token.
	marshalToken := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, relyingParty rp.RelyingParty) {
		handleOIDCCallback(w, r, tokens, handler)
	}
	mux.Handle("GET /auth/oidc/callback", rp.CodeExchangeHandler(marshalToken, provider))

	return mux, nil
}

func handleOIDCCallback(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], handler *Handler) {
	ctx := r.Context()
	claims := tokens.IDTokenClaims

	subject := claims.Subject
	email := claims.Email
	name := claims.Name
	if name == "" {
		name = email
	}

	q := dbgen.New(handler.DB)

	// Try to find existing user by OIDC subject.
	subjectStr := subject
	user, err := q.GetUserByOIDCSubject(ctx, &subjectStr)
	if err != nil {
		// User doesn't exist yet — create one.
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

	// Return the token as JSON. The frontend or CLI can extract it.
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"token":%q}`, raw)
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
