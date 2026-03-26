package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// CreateUserWithPassword creates a user with a bcrypt-hashed password.
func CreateUserWithPassword(ctx context.Context, pool *pgxpool.Pool, email, name, password string, isOwner bool) (dbgen.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("hash password: %w", err)
	}
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	q := dbgen.New(pool)
	return q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: pgtype.Text{String: string(hash), Valid: true},
		IsOwner:      isOwner,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}
