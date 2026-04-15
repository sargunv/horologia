package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/horologia/server/internal/auth"
)

func requireAuthenticatedDocs(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := authenticateRequest(r, pool); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func authenticateRequest(r *http.Request, pool *pgxpool.Pool) (*auth.User, error) {
	token := requestAuthToken(r)
	if token == "" {
		return nil, auth.ErrUnauthorized
	}

	return auth.AuthenticateBearerToken(r.Context(), pool, token, time.Now())
}

func requestAuthToken(r *http.Request) string {
	if authorization := r.Header.Get("Authorization"); authorization != "" {
		if token, ok := strings.CutPrefix(authorization, "Bearer "); ok {
			return token
		}
		return ""
	}

	if c, err := r.Cookie(sessionCookieName); err == nil {
		return c.Value
	}

	return ""
}
