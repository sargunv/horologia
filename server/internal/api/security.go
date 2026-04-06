package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// HandleBearerAuth validates the bearer token and enriches the context with the user.
func (h *Handler) HandleBearerAuth(ctx context.Context, operationName apigen.OperationName, t apigen.BearerAuth) (context.Context, error) {
	hash := auth.HashToken(t.Token)
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

	user := &auth.User{
		ID:      row.UserID,
		Email:   row.UserEmail,
		Name:    row.UserName,
		IsOwner: row.UserIsOwner,
	}

	if row.Kind == dbgen.AuthTokenKindApi {
		// Attribute API token identity for activity logging.
		user.Token = &auth.TokenInfo{ID: row.ID, Name: row.Name}
	} else {
		// Track session token hash so handlers can exclude the current session.
		user.SessionTokenHash = hash
	}

	return auth.ContextWithUser(ctx, user), nil
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
