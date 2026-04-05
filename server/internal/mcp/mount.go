package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// NewTransport creates an http.Handler for the MCP Streamable HTTP endpoint.
// The handler validates Bearer tokens (both API tokens and session tokens)
// and injects the authenticated user into the request context.
//
// Mount the returned handler at /mcp in your root mux:
//
//	mux.Handle("/mcp", mcp.NewTransport(pool))
func NewTransport(pool *pgxpool.Pool) http.Handler {
	s := NewServer()
	transport := mcpserver.NewStreamableHTTPServer(s)
	return bearerAuthMiddleware(pool, transport)
}

// bearerAuthMiddleware validates the Bearer token and injects auth.User into
// the context, or responds with 401 if the token is missing or invalid.
func bearerAuthMiddleware(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		hash := hashToken(token)
		q := dbgen.New(pool)
		row, err := q.GetAuthTokenByHash(r.Context(), hash)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		if row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		user := &auth.User{
			ID:      row.UserID,
			Email:   row.UserEmail,
			Name:    row.UserName,
			IsOwner: row.UserIsOwner,
		}

		if row.Kind == dbgen.AuthTokenKindApi {
			user.Token = &auth.TokenInfo{ID: row.ID, Name: row.Name}
		} else {
			user.SessionTokenHash = hash
		}

		next.ServeHTTP(w, r.WithContext(auth.ContextWithUser(r.Context(), user)))
	})
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
