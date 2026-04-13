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

// CookieAuthMiddleware reads the horologia_session cookie and injects it as a
// Bearer token header if no Authorization header is already present. This
// bridges cookie-based auth (web SPA) with ogen's bearer token validation.
func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			if c, err := r.Cookie(sessionCookieName); err == nil {
				r = r.Clone(r.Context())
				r.Header.Set("Authorization", "Bearer "+c.Value)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// WebLoginHandler returns an http.Handler for POST /auth/login.
// It validates credentials, creates a session token, sets an httpOnly cookie,
// and returns the user as JSON (without the raw token).
func WebLoginHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject non-JSON content types to prevent cross-site form POST (login CSRF).
		ct := r.Header.Get("Content-Type")
		if ct == "" || !strings.HasPrefix(strings.TrimSpace(strings.ToLower(ct)), "application/json") {
			writeJSONError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		ctx := r.Context()
		q := dbgen.New(handler.Pool)

		user, err := validatePassword(ctx, q, req.Email, req.Password)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid email or password")
			return
		}

		raw, err := createSessionToken(ctx, q, user.ID)
		if err != nil {
			handler.Log.ErrorContext(ctx, "login: create session", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		handler.setSessionCookie(w, raw)

		apiUser := userFromDB(user)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user": apiUser})
	})
}

// WebLogoutHandler returns an http.Handler for POST /auth/logout.
// It reads the session cookie, deletes the token from the DB, and clears the cookie.
func WebLogoutHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil {
			handler.clearSessionCookie(w)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		ctx := r.Context()
		hash := auth.HashToken(c.Value)
		q := dbgen.New(handler.Pool)
		_, _ = q.DeleteAuthTokenByHash(ctx, hash)

		handler.clearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})
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

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     sessionCookiePath,
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// AuthConfigHandler returns an http.Handler for GET /auth/config.
// It exposes public (no-auth) configuration the SPA needs before login.
func AuthConfigHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"oidc": map[string]any{
				"enabled":      handler.OIDCEnabled,
				"label":        handler.OIDCLabel,
				"autoRedirect": handler.OIDCAutoRedirect,
			},
			"password": map[string]any{
				"enabled": handler.PasswordAuthEnabled,
			},
		})
	})
}

// MountWebAuth wraps the base handler with cookie-based auth routes and middleware.
func MountWebAuth(base http.Handler, handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /auth/config", AuthConfigHandler(handler))
	if handler.PasswordAuthEnabled {
		mux.Handle("POST /auth/login", WebLoginHandler(handler))
	}
	if handler.OIDCLinkConsentEnabled {
		mux.Handle("GET /auth/link/pending", WebLinkPendingHandler(handler))
		mux.Handle("POST /auth/link", WebLinkHandler(handler))
	}
	mux.Handle("POST /auth/logout", WebLogoutHandler(handler))
	mux.Handle("/", CookieAuthMiddleware(base))
	return mux
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
