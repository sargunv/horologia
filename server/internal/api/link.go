package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/horologia/api/gen"
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

type pendingLinkStateContextKey struct{}

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

func pendingLinkStateFromContext(ctx context.Context) (pendingLinkState, bool) {
	state, ok := ctx.Value(pendingLinkStateContextKey{}).(pendingLinkState)
	return state, ok
}

func (h *Handler) clearPendingLinkCookie() *http.Cookie {
	return &http.Cookie{
		Name:     pendingLinkCookieName,
		Value:    "",
		Path:     pendingLinkCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func PendingLinkMiddleware(handler *Handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.LinkCookieHandler != nil {
			if state, err := readPendingLinkCookie(r, handler.LinkCookieHandler); err == nil {
				ctx := context.WithValue(r.Context(), pendingLinkStateContextKey{}, state)
				r = r.Clone(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) WebAuthLinkPending(ctx context.Context) (*apigen.AuthLinkPendingResponse, error) {
	if !h.OIDCLinkConsentEnabled {
		return nil, forbidden("oidc link consent is disabled")
	}

	state, ok := pendingLinkStateFromContext(ctx)
	if !ok {
		return nil, newAPIErrorResponse(http.StatusNotFound, "no pending link request")
	}

	return &apigen.AuthLinkPendingResponse{
		Email: state.Email,
		Name:  state.Name,
	}, nil
}

func (h *Handler) WebAuthLink(ctx context.Context, req *apigen.AuthLinkRequest) (*apigen.AuthLinkResponseHeaders, error) {
	if !h.OIDCLinkConsentEnabled {
		return nil, forbidden("oidc link consent is disabled")
	}

	state, ok := pendingLinkStateFromContext(ctx)
	if !ok {
		return nil, badRequest("link request not found or expired")
	}
	if err := defaultPasswordThrottle.beforeAttempt(ctx, state.Email); err != nil {
		return nil, err
	}

	// Validate password, link OIDC subject, and create session in one transaction.
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txq := dbgen.New(tx)

	// validatePassword handles no-password accounts with timing-safe sentinel comparison.
	user, err := validatePassword(ctx, txq, state.Email, req.Password)
	if errors.Is(err, errInvalidCredentials) {
		defaultPasswordThrottle.recordFailure(state.Email)
		return nil, badRequest("invalid password")
	}
	if err != nil {
		return nil, err
	}
	defaultPasswordThrottle.recordSuccess(state.Email)

	// Reject if the account was already linked to a different OIDC subject
	// between cookie issuance and now.
	if user.OidcSubject.Valid && user.OidcSubject.String != state.OIDCSubject {
		return nil, newAPIErrorResponse(http.StatusConflict, "account already linked to a different identity")
	}

	if err := txq.SetUserOIDCSubject(ctx, dbgen.SetUserOIDCSubjectParams{
		OidcSubject: pgtype.Text{String: state.OIDCSubject, Valid: true},
		UpdatedAt:   types.Timestamptz(time.Now()),
		ID:          user.ID,
	}); err != nil {
		h.Log.ErrorContext(ctx, "link: set oidc subject", "error", err)
		return nil, newAPIErrorResponse(http.StatusInternalServerError, "failed to link account")
	}

	raw, err := createSessionToken(ctx, txq, user.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	h.Log.InfoContext(ctx, "link: linked oidc subject", "email", state.Email, "subject", state.OIDCSubject)

	redirectTo := "/"
	if isValidRedirect(state.RedirectTo) {
		redirectTo = state.RedirectTo
	}

	return &apigen.AuthLinkResponseHeaders{
		SetCookie: []string{h.sessionCookie(raw).String(), h.clearPendingLinkCookie().String()},
		Response: apigen.AuthLinkResponse{
			RedirectTo: redirectTo,
		},
	}, nil
}

func newAPIErrorResponse(status int, message string) *apigen.ApiErrorStatusCode {
	return &apigen.ApiErrorStatusCode{
		StatusCode: status,
		Response: apigen.ApiError{
			Code:    httpStatusToCode(status),
			Message: message,
		},
	}
}
