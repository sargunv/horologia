package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/horologia/api/gen/go/ogen"
	"github.com/sargunv/horologia/server/internal/auth"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

func authenticateBearerToken(ctx context.Context, pool *pgxpool.Pool, token string) (*auth.User, error) {
	return auth.AuthenticateBearerToken(ctx, pool, token, time.Now())
}

func (h *Handler) authenticateToken(ctx context.Context, token string) (context.Context, error) {
	user, err := authenticateBearerToken(ctx, h.Pool, token)
	if errors.Is(err, auth.ErrUnauthorized) {
		return ctx, ogenerrors.ErrSkipServerSecurity
	}
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, sessionTokenContextKey{}, token)
	return auth.ContextWithUser(ctx, user), nil
}

// HandleBearerAuth validates the bearer token and enriches the context with the user.
func (h *Handler) HandleBearerAuth(ctx context.Context, operationName apigen.OperationName, t apigen.BearerAuth) (context.Context, error) {
	return h.authenticateToken(ctx, t.Token)
}

// generateToken creates a cryptographically random token and returns both
// the raw token (to give to the client) and the SHA-256 hash (to store in DB).
func generateToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, auth.HashToken(raw), nil
}

// createSessionToken generates a token, stores it in the DB, and returns the raw value.
// The token expires after sessionMaxAge seconds, matching the session cookie lifetime.
func createSessionToken(ctx context.Context, q *dbgen.Queries, userID int64) (string, error) {
	raw, hash, err := generateToken()
	if err != nil {
		return "", err
	}
	now := time.Now()
	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hash,
		Name:      "",
		Kind:      dbgen.AuthTokenKindSession,
		ExpiresAt: types.Timestamptz(now.Add(sessionMaxAge * time.Second)),
		CreatedAt: types.Timestamptz(now),
	})
	return raw, err
}
