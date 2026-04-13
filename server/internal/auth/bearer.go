package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
)

var ErrUnauthorized = errors.New("unauthorized")

func authenticateToken(ctx context.Context, db dbgen.DBTX, token string, now time.Time) (*User, error) {
	hash := HashToken(token)

	row, err := dbgen.New(db).GetAuthTokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}

	if row.ExpiresAt.Valid && now.After(row.ExpiresAt.Time) {
		return nil, ErrUnauthorized
	}

	user := &User{
		ID:      row.UserID,
		Email:   row.UserEmail,
		Name:    row.UserName,
		IsOwner: row.UserIsOwner,
	}

	switch row.Kind {
	case dbgen.AuthTokenKindSession:
		user.SessionTokenHash = hash
	case dbgen.AuthTokenKindApi, dbgen.AuthTokenKindOauthAccess, dbgen.AuthTokenKindOauthRefresh:
		tokenInfo := &TokenInfo{
			ID:     row.ID,
			Name:   row.Name,
			Kind:   row.Kind,
			Scopes: append([]string(nil), row.OauthScopes...),
		}
		if row.OauthClientID.Valid {
			tokenInfo.ClientID = row.OauthClientID.String
		}
		if row.OauthResource.Valid {
			tokenInfo.Resource = row.OauthResource.String
		}
		user.Token = tokenInfo
	default:
		return nil, ErrUnauthorized
	}

	return user, nil
}

// AuthenticateBearerToken resolves the raw bearer token against the auth token
// table and returns the authenticated user context for downstream handlers.
// Refresh tokens are not valid bearer credentials for API or MCP access.
func AuthenticateBearerToken(ctx context.Context, db dbgen.DBTX, token string, now time.Time) (*User, error) {
	user, err := authenticateToken(ctx, db, token, now)
	if err != nil {
		return nil, err
	}
	if user.Token != nil && user.Token.Kind == dbgen.AuthTokenKindOauthRefresh {
		return nil, ErrUnauthorized
	}
	return user, nil
}

// AuthenticateRefreshToken resolves and validates a refresh token for the
// token endpoint. Refresh tokens are intentionally not accepted by the normal
// bearer-auth path used by API and MCP requests.
func AuthenticateRefreshToken(ctx context.Context, db dbgen.DBTX, token string, now time.Time) (*User, error) {
	user, err := authenticateToken(ctx, db, token, now)
	if err != nil {
		return nil, err
	}
	if user.Token == nil || user.Token.Kind != dbgen.AuthTokenKindOauthRefresh {
		return nil, ErrUnauthorized
	}
	return user, nil
}
