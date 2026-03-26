package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

type contextKey int

const (
	contextKeyUser contextKey = iota
)

// AuthUser is the authenticated user attached to the request context.
type AuthUser struct {
	ID      int64
	Email   string
	Name    string
	IsOwner bool
}

// UserFromContext retrieves the authenticated user from the context.
// Returns nil if unauthenticated.
func UserFromContext(ctx context.Context) *AuthUser {
	u, _ := ctx.Value(contextKeyUser).(*AuthUser)
	return u
}

// ContextWithUser returns a new context with the given user attached.
func ContextWithUser(ctx context.Context, u *AuthUser) context.Context {
	return context.WithValue(ctx, contextKeyUser, u)
}

// HandleBearerAuth validates the bearer token and enriches the context with the user.
func (h *Handler) HandleBearerAuth(ctx context.Context, operationName apigen.OperationName, t apigen.BearerAuth) (context.Context, error) {
	hash := hashToken(t.Token)
	q := dbgen.New(h.Pool)
	row, err := q.GetAuthTokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ctx, ogenerrors.ErrSkipServerSecurity
	}
	if err != nil {
		return ctx, err
	}

	// Check expiration.
	if row.ExpiresAt.Valid && time.Now().After(row.ExpiresAt.Time) {
		return ctx, ogenerrors.ErrSkipServerSecurity
	}

	user := &AuthUser{
		ID:      row.UserID,
		Email:   row.UserEmail,
		Name:    row.UserName,
		IsOwner: row.UserIsOwner,
	}
	return ContextWithUser(ctx, user), nil
}

// generateToken creates a cryptographically random token and returns both
// the raw token (to give to the client) and the SHA-256 hash (to store in DB).
func generateToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw), nil
}

// hashToken returns the hex-encoded SHA-256 of the given token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// createSessionToken generates a token, stores it in the DB, and returns the raw value.
func createSessionToken(ctx context.Context, q *dbgen.Queries, userID int64) (string, error) {
	raw, hash, err := generateToken()
	if err != nil {
		return "", err
	}
	_, err = q.CreateAuthToken(ctx, dbgen.CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hash,
		Name:      "",
		Kind:      dbgen.AuthTokenKindSession,
		CreatedAt: timeToTS(time.Now()),
	})
	return raw, err
}
