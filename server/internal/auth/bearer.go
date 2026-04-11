package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

var ErrUnauthorized = errors.New("unauthorized")

// AuthenticateBearerToken resolves the raw bearer token against the auth token
// table and returns the authenticated user context for downstream handlers.
func AuthenticateBearerToken(ctx context.Context, db dbgen.DBTX, token string, now time.Time) (*User, error) {
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
	default:
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
	}

	return user, nil
}
