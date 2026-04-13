package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	zhttp "github.com/zitadel/oidc/v3/pkg/http"

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

const (
	pendingLinkCookieName = "horologia_oidc_link" //nolint:gosec // cookie name, not a credential
	pendingLinkCookiePath = "/"
	pendingLinkMaxAge     = 300 // 5 minutes
)

// pendingLinkState holds the data stored in the encrypted pending-link cookie.
type pendingLinkState struct {
	OIDCSubject string `json:"sub"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	RedirectTo  string `json:"redirect,omitempty"`
	ExpiresAt   int64  `json:"exp"`
}

var errNoPendingLink = errors.New("no pending link cookie")

// NewLinkCookieHandler creates a zitadel CookieHandler for the pending-link
// cookie using fresh ephemeral keys. The cookie uses path "/" to ensure it is
// accessible to both the OIDC callback (which sets it) and the link endpoints
// (which read it), consistent with other cookies in the application.
func NewLinkCookieHandler(secureCookies bool) (*zhttp.CookieHandler, error) {
	hashKey := make([]byte, 32)
	if _, err := rand.Read(hashKey); err != nil {
		return nil, fmt.Errorf("generate link cookie hash key: %w", err)
	}
	encKey := make([]byte, 32)
	if _, err := rand.Read(encKey); err != nil {
		return nil, fmt.Errorf("generate link cookie encryption key: %w", err)
	}
	opts := []zhttp.CookieHandlerOpt{
		zhttp.WithPath(pendingLinkCookiePath),
		zhttp.WithMaxAge(pendingLinkMaxAge),
		zhttp.WithSameSite(http.SameSiteLaxMode),
	}
	if !secureCookies {
		opts = append(opts, zhttp.WithUnsecure())
	}
	return zhttp.NewCookieHandler(hashKey, encKey, opts...), nil
}

func setPendingLinkCookie(w http.ResponseWriter, ch *zhttp.CookieHandler, state pendingLinkState) error {
	state.ExpiresAt = time.Now().Add(pendingLinkMaxAge * time.Second).Unix()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal pending link state: %w", err)
	}
	return ch.SetCookie(w, pendingLinkCookieName, string(data))
}

func readPendingLinkCookie(r *http.Request, ch *zhttp.CookieHandler) (pendingLinkState, error) {
	value, err := ch.CheckCookie(r, pendingLinkCookieName)
	if err != nil {
		return pendingLinkState{}, errNoPendingLink
	}
	var state pendingLinkState
	if err := json.Unmarshal([]byte(value), &state); err != nil {
		return pendingLinkState{}, errNoPendingLink
	}
	if time.Now().Unix() > state.ExpiresAt {
		return pendingLinkState{}, errNoPendingLink
	}
	return state, nil
}

// WebLinkPendingHandler returns an http.Handler for GET /auth/link/pending.
// It reads the pending-link cookie and returns the email/name for the SPA to display.
func WebLinkPendingHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state, err := readPendingLinkCookie(r, handler.LinkCookieHandler)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "no pending link request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"email": state.Email,
			"name":  state.Name,
		})
	})
}

// WebLinkHandler returns an http.Handler for POST /auth/link.
// It validates the user's password, links the OIDC subject, and creates a session.
func WebLinkHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject non-JSON content types to prevent cross-site form POST (CSRF).
		ct := r.Header.Get("Content-Type")
		if ct == "" || !strings.HasPrefix(strings.TrimSpace(strings.ToLower(ct)), "application/json") {
			writeJSONError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}

		var req struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		ctx := r.Context()

		state, err := readPendingLinkCookie(r, handler.LinkCookieHandler)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "link request not found or expired")
			return
		}

		// Validate password, link OIDC subject, and create session in one transaction.
		tx, err := handler.Pool.Begin(ctx)
		if err != nil {
			handler.Log.ErrorContext(ctx, "link: begin tx", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		txq := dbgen.New(tx)

		// validatePassword handles no-password accounts with timing-safe sentinel comparison.
		user, err := validatePassword(ctx, txq, state.Email, req.Password)
		if errors.Is(err, errInvalidCredentials) {
			writeJSONError(w, http.StatusBadRequest, "invalid password")
			return
		}
		if err != nil {
			handler.Log.ErrorContext(ctx, "link: validate password", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Reject if the account was already linked to a different OIDC subject
		// between cookie issuance and now.
		if user.OidcSubject.Valid && user.OidcSubject.String != state.OIDCSubject {
			writeJSONError(w, http.StatusConflict, "account already linked to a different identity")
			return
		}

		if err := txq.SetUserOIDCSubject(ctx, dbgen.SetUserOIDCSubjectParams{
			OidcSubject: pgtype.Text{String: state.OIDCSubject, Valid: true},
			UpdatedAt:   types.Timestamptz(time.Now()),
			ID:          user.ID,
		}); err != nil {
			handler.Log.ErrorContext(ctx, "link: set oidc subject", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to link account")
			return
		}

		raw, err := createSessionToken(ctx, txq, user.ID)
		if err != nil {
			handler.Log.ErrorContext(ctx, "link: create session", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		if err := tx.Commit(ctx); err != nil {
			handler.Log.ErrorContext(ctx, "link: commit tx", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		handler.setSessionCookie(w, raw)
		handler.LinkCookieHandler.DeleteCookie(w, pendingLinkCookieName)

		handler.Log.InfoContext(ctx, "link: linked oidc subject", "email", state.Email, "subject", state.OIDCSubject)

		redirectTo := "/"
		if isValidRedirect(state.RedirectTo) {
			redirectTo = state.RedirectTo
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"linked":     true,
			"redirectTo": redirectTo,
		})
	})
}
