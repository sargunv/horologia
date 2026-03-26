package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// CreateUserWithPassword creates a user with a bcrypt-hashed password.
func CreateUserWithPassword(ctx context.Context, db database.DB, email, name, password string, isOwner bool) (dbgen.User, error) {
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
