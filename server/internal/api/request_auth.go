package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/horologia/server/internal/auth"
)

func requireAuthenticatedDocs(pool *pgxpool.Pool, publicURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r, pool, publicURL); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authenticateRequest(r *http.Request, pool *pgxpool.Pool, publicURL string) (*auth.User, error) {
	if token, ok := requestBearerToken(r); ok {
		return auth.AuthenticateBearerToken(r.Context(), pool, token, time.Now())
	}

	token, ok := requestSessionToken(r)
	if !ok || !sameOriginRequest(r, publicURL) {
		return nil, auth.ErrUnauthorized
	}

	return auth.AuthenticateBearerToken(r.Context(), pool, token, time.Now())
}

func requestBearerToken(r *http.Request) (string, bool) {
	if authorization := r.Header.Get("Authorization"); authorization != "" {
		if len(authorization) > 7 && strings.EqualFold(authorization[:7], "Bearer ") {
			return authorization[7:], true
		}
		return "", false
	}
	return "", false
}

func requestSessionToken(r *http.Request) (string, bool) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		return c.Value, true
	}

	return "", false
}
