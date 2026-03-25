package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// sentinelHash is a pre-computed bcrypt hash used to prevent timing-based
// email enumeration. When the email is not found or has no password, we still
// run a bcrypt comparison against this hash so the response time is constant.
var sentinelHash, _ = bcrypt.GenerateFromPassword([]byte("sentinel"), bcrypt.DefaultCost)

const (
	sessionCookieName = "tend_session"
	sessionCookiePath = "/"
	sessionMaxAge     = 60 * 60 * 24 * 30 // 30 days
)

// CookieAuthMiddleware reads the tend_session cookie and injects it as a
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

// WebLoginHandler returns an http.Handler for POST /auth/web-login.
// It validates credentials, creates a session token, sets an httpOnly cookie,
// and returns the user as JSON (without the raw token).
func WebLoginHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		ctx := r.Context()
		q := dbgen.New(handler.DB)

		user, err := validatePassword(ctx, q, req.Email, req.Password)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid email or password")
			return
		}

		raw, err := createSessionToken(ctx, q, user.ID)
		if err != nil {
			handler.Log.ErrorContext(ctx, "web-login: create session", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		setSessionCookie(w, raw)

		apiUser := userFromDB(user)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"user": apiUser})
	})
}

// WebLogoutHandler returns an http.Handler for POST /auth/web-logout.
// It reads the session cookie, deletes the token from the DB, and clears the cookie.
func WebLogoutHandler(handler *Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookieName)
		if err != nil {
			clearSessionCookie(w)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		ctx := r.Context()
		hash := hashToken(c.Value)
		q := dbgen.New(handler.DB)
		_, _ = q.DeleteAuthTokenByHash(ctx, hash)

		clearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	})
}

// validatePassword checks a user's email and password. Returns the user on
// success or an error on failure. Uses timing-safe comparison to prevent
// email enumeration.
func validatePassword(ctx context.Context, q *dbgen.Queries, email, password string) (dbgen.User, error) {
	user, err := q.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(password))
		return dbgen.User{}, errors.New("invalid credentials")
	}
	if err != nil {
		return dbgen.User{}, err
	}

	if user.PasswordHash == nil {
		// OIDC-only user — no password login allowed. Run bcrypt against
		// the sentinel hash to prevent timing-based enumeration.
		_ = bcrypt.CompareHashAndPassword(sentinelHash, []byte(password))
		return dbgen.User{}, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return dbgen.User{}, errors.New("invalid credentials")
	}

	return user, nil
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     sessionCookiePath,
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     sessionCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// MountWebAuth wraps the base handler with cookie-based auth routes and middleware.
func MountWebAuth(base http.Handler, handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /auth/web-login", WebLoginHandler(handler))
	mux.Handle("POST /auth/web-logout", WebLogoutHandler(handler))
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
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
