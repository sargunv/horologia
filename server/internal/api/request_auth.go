package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/horologia/server/internal/auth"
)

func requireAuthenticatedDocs(pool *pgxpool.Pool, publicURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := checkAuthenticatedRequest(r, pool, publicURL); err != nil {
			if errors.Is(err, auth.ErrUnauthorized) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func checkAuthenticatedRequest(r *http.Request, pool *pgxpool.Pool, publicURL string) error {
	if token, ok := requestBearerToken(r); ok {
		_, err := auth.AuthenticateBearerToken(r.Context(), pool, token, time.Now())
		return err
	}

	token, ok := requestSessionToken(r)
	if !ok || !sameOriginRequest(r, publicURL) {
		return auth.ErrUnauthorized
	}

	_, err := auth.AuthenticateBearerToken(r.Context(), pool, token, time.Now())
	return err
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
