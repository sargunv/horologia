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

// CreateUserWithPassword creates a user with a bcrypt-hashed password.
// The checker parameter is used for HIBP breach checking; nil disables it.
// TODO: Future password-change and user-management endpoints should call
// pwdcheck.Validate before any bcrypt hashing.
func CreateUserWithPassword(ctx context.Context, db database.DB, email, name, password string, isOwner bool, checker pwdcheck.Checker) (dbgen.User, error) {
	if err := pwdcheck.Validate(ctx, password, checker); err != nil {
		return dbgen.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("hash password: %w", err)
	}
	now := types.Timestamptz(time.Now())

	q := dbgen.New(db)
	return q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: pgtype.Text{String: string(hash), Valid: true},
		IsOwner:      isOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

// CreateUserWithoutPassword creates a user with no password hash, suitable
// for OIDC-only deployments where password auth is disabled.
func CreateUserWithoutPassword(ctx context.Context, db database.DB, email, name string, isOwner bool) (dbgen.User, error) {
	now := types.Timestamptz(time.Now())

	q := dbgen.New(db)
	return q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: pgtype.Text{Valid: false},
		IsOwner:      isOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}
