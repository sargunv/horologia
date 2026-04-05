package mcp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	mcpgen "github.com/sargunv/tend/server/internal/mcp/gen"
)

// NewTransport creates an http.Handler for the MCP Streamable HTTP endpoint.
// The handler validates Bearer tokens (both API tokens and session tokens)
// and injects the authenticated user into the request context.
//
// Returns 401 Unauthorized if the Authorization header is missing, the token
// is not found in the database, or the token has expired. On success the
// authenticated user is available via auth.UserFromContext(ctx).
//
// Mount the returned handler at /mcp in your root mux:
//
//	mux.Handle("/mcp", mcp.NewTransport(pool, handler))
func NewTransport(pool *pgxpool.Pool, h mcpgen.Handlers) http.Handler {
	s := mcpserver.NewMCPServer("Tend", "0.1.0")
	mcpgen.RegisterTools(s, h)
	transport := mcpserver.NewStreamableHTTPServer(s)
	return bearerAuthMiddleware(pool, transport)
}

// bearerAuthMiddleware validates the Bearer token and injects auth.User into
// the context, or responds with 401 if the token is missing or invalid.
func bearerAuthMiddleware(pool *pgxpool.Pool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		hash := auth.HashToken(token)
		q := dbgen.New(pool)
		row, err := q.GetAuthTokenByHash(r.Context(), hash)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time) {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
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

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		slog.Error("mcp: writeJSONError: failed to write response", "error", err)
	}
}
