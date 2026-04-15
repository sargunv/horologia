package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	apigen "github.com/sargunv/horologia/api/gen"
	"github.com/sargunv/horologia/server/internal/auth"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
)

// sentinelHash is a pre-computed bcrypt hash used to prevent timing-based
// email enumeration. When the email is not found or has no password, we still
// run a bcrypt comparison against this hash so the response time is constant.
var sentinelHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("sentinel"), bcrypt.DefaultCost)
	if err != nil {
		panic("bcrypt: generate sentinel hash: " + err.Error())
	}
	return h
}()

var errInvalidCredentials = errors.New("invalid credentials")

const (
	sessionCookieName = "horologia_session"
	sessionCookiePath = "/"
	sessionMaxAge     = 60 * 60 * 24 * 30 // 30 days
)

type (
	sessionTokenContextKey      struct{}
	authHeaderPresentContextKey struct{}
)

// CookieAuthMiddleware reads the horologia_session cookie and exposes it on the
// request context for handlers that need the raw session token.
func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			ctx := context.WithValue(r.Context(), authHeaderPresentContextKey{}, true)
			r = r.Clone(ctx)
		}

		if c, err := r.Cookie(sessionCookieName); err == nil {
			ctx := context.WithValue(r.Context(), sessionTokenContextKey{}, c.Value)
			r = r.Clone(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

func sessionTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(sessionTokenContextKey{}).(string)
	return token, ok
}

func authHeaderPresent(ctx context.Context) bool {
	present, _ := ctx.Value(authHeaderPresentContextKey{}).(bool)
	return present
}

func (h *Handler) sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     sessionCookiePath,
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

// validatePassword checks a user's email and password. Returns the user on
// success or an error on failure. Uses timing-safe comparison to prevent
// email enumeration.
func validatePassword(ctx context.Context, q *dbgen.Queries, email, password string) (dbgen.User, error) {
	user, err := q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(password))
		return dbgen.User{}, errInvalidCredentials
	}
	if err != nil {
		return dbgen.User{}, err
	}

	if !user.PasswordHash.Valid {
		// OIDC-only user — no password login allowed. Run bcrypt against
		// the sentinel hash to prevent timing-based enumeration.
		_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(password))
		return dbgen.User{}, errInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password)); err != nil {
		return dbgen.User{}, errInvalidCredentials
	}

	return user, nil
}

func (h *Handler) clearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, h.sessionCookie(token))
}

func (h *Handler) clearSessionCookieHeader(w http.ResponseWriter) {
	http.SetCookie(w, h.clearSessionCookie())
}

func (h *Handler) WebAuthConfig(ctx context.Context) (*apigen.AuthConfig, error) {
	return &apigen.AuthConfig{
		Oidc: apigen.AuthConfigOIDC{
			Enabled:      h.OIDCEnabled,
			Label:        h.OIDCLabel,
			AutoRedirect: h.OIDCAutoRedirect,
		},
		Password: apigen.AuthConfigPassword{
			Enabled: h.PasswordAuthEnabled,
		},
	}, nil
}

func (h *Handler) WebAuthLogin(ctx context.Context, req *apigen.AuthLoginRequest) (*apigen.AuthLoginResponseHeaders, error) {
	if !h.PasswordAuthEnabled {
		return nil, forbidden("password authentication is disabled")
	}
	if err := defaultPasswordThrottle.beforeAttempt(ctx, req.Email); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)
	user, err := validatePassword(ctx, q, req.Email, req.Password)
	if err != nil {
		defaultPasswordThrottle.recordFailure(req.Email)
		return nil, badRequest("invalid email or password")
	}
	defaultPasswordThrottle.recordSuccess(req.Email)

	raw, err := createSessionToken(ctx, q, user.ID)
	if err != nil {
		return nil, err
	}

	apiUser := userFromDB(user)
	return &apigen.AuthLoginResponseHeaders{
		SetCookie: h.sessionCookie(raw).String(),
		Response: apigen.AuthLoginResponse{
			User: *apiUser,
		},
	}, nil
}

func (h *Handler) WebAuthLogout(ctx context.Context) (*apigen.WebAuthLogoutNoContent, error) {
	if token, ok := sessionTokenFromContext(ctx); ok {
		hash := auth.HashToken(token)
		q := dbgen.New(h.Pool)
		_, _ = q.DeleteAuthTokenByHash(ctx, hash)
	}

	return &apigen.WebAuthLogoutNoContent{SetCookie: h.clearSessionCookie().String()}, nil
}

func httpStatusToCode(status int) string {
	text := http.StatusText(status)
	if text == "" {
		return "internal_error"
	}
	return strings.ToLower(strings.ReplaceAll(text, " ", "_"))
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"code":    httpStatusToCode(status),
		"message": message,
	}); err != nil {
		slog.Error("writeJSONError: failed to write response", "error", err)
	}
}

// MountWebAuth wraps the base handler with cookie-backed request context.
func MountWebAuth(base http.Handler, handler *Handler) http.Handler {
	return PendingLinkMiddleware(handler, CookieAuthMiddleware(base))
}
