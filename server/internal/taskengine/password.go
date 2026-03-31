package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/pwdcheck"
	"github.com/sargunv/tend/server/internal/types"
)

// SetUserPassword validates and sets a user's password hash.
func SetUserPassword(ctx context.Context, db database.DB, userID int64, password string, checker pwdcheck.Checker, now time.Time) error {
	if err := pwdcheck.Validate(ctx, password, checker); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	q := dbgen.New(db)
	return q.UpdateUserPasswordHash(ctx, dbgen.UpdateUserPasswordHashParams{
		PasswordHash: pgtype.Text{String: string(hash), Valid: true},
		UpdatedAt:    types.Timestamptz(now),
		ID:           userID,
	})
}

// ClearUserPassword removes a user's password hash, making them OIDC-only.
func ClearUserPassword(ctx context.Context, db database.DB, userID int64, now time.Time) error {
	q := dbgen.New(db)
	return q.UpdateUserPasswordHash(ctx, dbgen.UpdateUserPasswordHashParams{
		PasswordHash: pgtype.Text{Valid: false},
		UpdatedAt:    types.Timestamptz(now),
		ID:           userID,
	})
}
