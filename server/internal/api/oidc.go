package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/zitadel/oidc/v3/pkg/client/rp"
	zhttp "github.com/zitadel/oidc/v3/pkg/http"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

const oidcRedirectCookieName = "tend_oidc_redirect" //nolint:gosec // cookie name, not a credential

// isValidRedirect checks that a redirect path is safe (relative, no open redirect).
func isValidRedirect(path string) bool {
	return path != "" && strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")
}

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
	// Generate keys for HMAC authentication and AES encryption of OIDC state cookies.
	hashKey := make([]byte, 32)
	if _, err := rand.Read(hashKey); err != nil {
		return nil, fmt.Errorf("generate oidc hash key: %w", err)
	}
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("generate oidc encryption key: %w", err)
	}
	cookieOpts := []zhttp.CookieHandlerOpt{}
	if !handler.SecureCookies {
		cookieOpts = append(cookieOpts, zhttp.WithUnsecure())
	}
	cookieHandler := zhttp.NewCookieHandler(hashKey, encKey, cookieOpts...)

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
	// Wraps AuthURLHandler to preserve an optional ?redirect= param in a
	// short-lived cookie so the callback can send the user to the right page.
	// Note: the redirect cookie has the same Safari/ITP limitation as the state
	// cookie — it may not survive the round-trip through a dev proxy. In dev
	// mode the redirect falls back to "/" which is acceptable.
	authURLHandler := rp.AuthURLHandler(func() string {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			panic("generate oidc state: " + err.Error())
		}
		return hex.EncodeToString(b)
	}, provider)
	mux.Handle("GET /auth/oidc", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rd := r.URL.Query().Get("redirect"); isValidRedirect(rd) {
			http.SetCookie(w, &http.Cookie{
				Name:     oidcRedirectCookieName,
				Value:    rd,
				Path:     "/",
				MaxAge:   300,
				HttpOnly: true,
				Secure:   handler.SecureCookies,
				SameSite: http.SameSiteLaxMode,
			})
		}
		authURLHandler.ServeHTTP(w, r)
	}))

	// GET /auth/oidc/callback → exchange code, find/create user, issue token.
	marshalToken := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, relyingParty rp.RelyingParty) {
		handler.handleOIDCCallback(w, r, tokens, relyingParty)
	}
	mux.Handle("GET /auth/oidc/callback", rp.CodeExchangeHandler(marshalToken, provider))

	return mux, nil
}

func (h *Handler) handleOIDCCallback(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], relyingParty rp.RelyingParty) {
	ctx := r.Context()

	subject := tokens.IDTokenClaims.Subject

	// The ID token may not contain email/profile claims (per OIDC spec,
	// they go in userinfo when an access token is issued). Fetch from userinfo.
	info, err := rp.Userinfo[*oidc.UserInfo](ctx, tokens.AccessToken, tokens.TokenType, subject, relyingParty)
	if err != nil {
		h.Log.ErrorContext(ctx, "oidc: userinfo request failed", "error", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	email := info.Email
	if email == "" {
		email = tokens.IDTokenClaims.Email
	}
	if email == "" {
		h.Log.ErrorContext(ctx, "oidc: no email from provider", "subject", subject)
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

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		h.Log.ErrorContext(ctx, "oidc: begin tx", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	subjectText := pgtype.Text{String: subject, Valid: true}

	// Try to find existing user by OIDC subject.
	user, err := q.GetUserByOIDCSubject(ctx, subjectText)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		h.Log.ErrorContext(ctx, "oidc: get user", "error", err)
		http.Error(w, "failed to look up user", http.StatusInternalServerError)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		// No user with this OIDC subject. Try matching by email to link
		// an existing password-created account.
		// TODO: Auto-linking by email means any trusted OIDC provider can claim
		// an existing account without user consent. Consider requiring explicit
		// user approval before linking OIDC identities to password-based accounts.
		user, err = q.GetUserByEmail(ctx, email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			h.Log.ErrorContext(ctx, "oidc: get user by email", "error", err)
			http.Error(w, "failed to look up user", http.StatusInternalServerError)
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			// Completely new user — create one.
			ts := time.Now()
			tstz := types.Timestamptz(ts)
			user, err = q.CreateUser(ctx, dbgen.CreateUserParams{
				Email:       email,
				Name:        name,
				OidcSubject: subjectText,
				CreatedAt:   tstz,
				UpdatedAt:   tstz,
			})
			if err != nil {
				h.Log.ErrorContext(ctx, "oidc: create user", "error", err)
				http.Error(w, "failed to create user", http.StatusInternalServerError)
				return
			}
			h.Log.InfoContext(ctx, "oidc: created user", "email", email, "subject", subject)
		} else {
			// Existing user found by email — link the OIDC subject.
			if err := q.SetUserOIDCSubject(ctx, dbgen.SetUserOIDCSubjectParams{
				OidcSubject: subjectText,
				UpdatedAt:   types.Timestamptz(time.Now()),
				ID:          user.ID,
			}); err != nil {
				h.Log.ErrorContext(ctx, "oidc: link user", "error", err)
				http.Error(w, "failed to link user", http.StatusInternalServerError)
				return
			}
			h.Log.InfoContext(ctx, "oidc: linked user", "email", email, "subject", subject)
		}
	}

	// Generate a session token.
	raw, err := createSessionToken(ctx, q, user.ID)
	if err != nil {
		h.Log.ErrorContext(ctx, "oidc: create session", "error", err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		h.Log.ErrorContext(ctx, "oidc: commit tx", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, raw)

	// Read and clear the OIDC redirect cookie.
	redirectTo := "/"
	if c, err := r.Cookie(oidcRedirectCookieName); err == nil && isValidRedirect(c.Value) {
		redirectTo = c.Value
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcRedirectCookieName,
		Value:    "",
		Path:     "/auth/oidc/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// MountOIDC wraps an ogen handler with optional OIDC routes.
// If oidcHandler is nil, returns the base handler unchanged.
func MountOIDC(base http.Handler, oidcHandler http.Handler, log *slog.Logger) http.Handler {
	if oidcHandler == nil {
		return base
	}
	mux := http.NewServeMux()
	mux.Handle("/auth/oidc", oidcHandler)
	mux.Handle("/auth/oidc/", oidcHandler)
	mux.Handle("/", base)
	log.Info("OIDC enabled")
	return mux
}
