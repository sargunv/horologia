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

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

const oidcRedirectCookieName = "horologia_oidc_redirect" //nolint:gosec // cookie name, not a credential

// isValidRedirect checks that a redirect path is safe (relative, no open redirect).
func isValidRedirect(path string) bool {
	return path != "" && strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//")
}

// readAndClearRedirectCookie reads the OIDC redirect cookie value and clears it.
// Returns the redirect path, or fallback if the cookie is missing or invalid.
func (h *Handler) readAndClearRedirectCookie(w http.ResponseWriter, r *http.Request, fallback string) string {
	redirectTo := fallback
	if c, err := r.Cookie(oidcRedirectCookieName); err == nil && isValidRedirect(c.Value) {
		redirectTo = c.Value
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oidcRedirectCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	return redirectTo
}

// OIDCConfig holds the configuration for the OIDC relying party.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	PublicURL    string
}

// NewOIDCHandler creates an http.Handler that handles the OIDC authorization
// code flow. It registers two routes:
//
//	GET /app/auth/oidc          → redirects to the OIDC provider
//	GET /app/auth/oidc/callback → handles the callback, creates a user+token
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
	var cookieOpts []zhttp.CookieHandlerOpt
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
		cfg.PublicURL+"/app/auth/oidc/callback",
		[]string{"openid", "email", "profile"},
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("create oidc relying party: %w", err)
	}

	mux := http.NewServeMux()

	// GET /app/auth/oidc → redirect to IdP.
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
	mux.Handle("GET /app/auth/oidc", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// GET /app/auth/oidc/callback → exchange code, find/create user, issue token.
	marshalToken := func(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, relyingParty rp.RelyingParty) {
		handler.handleOIDCCallback(w, r, tokens, relyingParty)
	}
	mux.Handle("GET /app/auth/oidc/callback", rp.CodeExchangeHandler(marshalToken, provider))

	return mux, nil
}

func (h *Handler) handleOIDCCallback(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], relyingParty rp.RelyingParty) {
	ctx := r.Context()
	now := time.Now()

	subject := tokens.IDTokenClaims.Subject

	// The ID token may not contain email/profile claims (per OIDC spec,
	// they go in userinfo when an access token is issued). Fetch from userinfo.
	// Use tokens.Type() instead of tokens.TokenType to normalize the token type
	// (e.g. "bearer" → "Bearer") per RFC 6750 case-insensitivity requirements.
	info, err := rp.Userinfo[*oidc.UserInfo](ctx, tokens.AccessToken, tokens.Type(), subject, relyingParty)
	if err != nil {
		h.Log.ErrorContext(ctx, "oidc: userinfo request failed", "error", err)
		http.Error(w, "failed to fetch user info", http.StatusInternalServerError)
		return
	}

	emailVerified := info.EmailVerified
	if !emailVerified {
		emailVerified = tokens.IDTokenClaims.EmailVerified
	}
	if !emailVerified {
		h.Log.ErrorContext(ctx, "oidc: email not verified", "subject", subject)
		http.Error(w, "OIDC provider email is not verified", http.StatusBadRequest)
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
		// an existing account.
		user, err = q.GetUserByEmail(ctx, email)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			h.Log.ErrorContext(ctx, "oidc: get user by email", "error", err)
			http.Error(w, "failed to look up user", http.StatusInternalServerError)
			return
		}
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// Completely new user — create one.
			tstz := types.Timestamptz(now)
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
		case h.OIDCLinkConsentEnabled:
			// Existing user found by email — require consent before linking.
			// The deferred tx.Rollback will clean up; linking happens in POST /app/auth/link.
			redirectTo := h.readAndClearRedirectCookie(w, r, "")

			if err := setPendingLinkCookie(w, h.LinkCookieHandler, pendingLinkState{
				OIDCSubject: subject,
				Email:       email,
				Name:        name,
				RedirectTo:  redirectTo,
			}); err != nil {
				h.Log.ErrorContext(ctx, "oidc: set pending link cookie", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}

			h.Log.InfoContext(ctx, "oidc: consent required for link", "email", email, "subject", subject)
			http.Redirect(w, r, "/link-account", http.StatusSeeOther)
			return
		default:
			// Existing user found by email — link the OIDC subject directly.
			// TODO: This unconditionally overwrites any existing OIDC subject on the
			// matched user. If the user already has a different OIDC subject linked,
			// a new OIDC identity with the same email can silently take over the account.
			// This is acceptable when the admin has explicitly disabled consent (trusting
			// the OIDC provider), but should be revisited when multi-provider support
			// is added — at minimum, reject subject overwrites or require consent.
			if err := q.SetUserOIDCSubject(ctx, dbgen.SetUserOIDCSubjectParams{
				OidcSubject: subjectText,
				UpdatedAt:   types.Timestamptz(now),
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

	redirectTo := h.readAndClearRedirectCookie(w, r, "/")
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

// MountOIDC wraps an ogen handler with optional OIDC routes.
// If oidcHandler is nil, returns the base handler unchanged.
func MountOIDC(base http.Handler, oidcHandler http.Handler, log *slog.Logger) http.Handler {
	if oidcHandler == nil {
		return base
	}
	mux := http.NewServeMux()
	mux.Handle("/app/auth/oidc", oidcHandler)
	mux.Handle("/app/auth/oidc/", oidcHandler)
	mux.Handle("/", base)
	log.Info("OIDC enabled")
	return mux
}
