package api

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// CreateUserWithPassword creates a user with a bcrypt-hashed password.
func CreateUserWithPassword(ctx context.Context, db *sql.DB, email, name, password string, isOwner bool) (dbgen.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return dbgen.User{}, fmt.Errorf("hash password: %w", err)
	}
	hashStr := string(hash)
	now := types.Now()

	var ownerFlag int64
	if isOwner {
		ownerFlag = 1
	}

	q := dbgen.New(db)
	return q.CreateUser(ctx, dbgen.CreateUserParams{
		Email:        email,
		Name:         name,
		PasswordHash: &hashStr,
		IsOwner:      ownerFlag,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}
