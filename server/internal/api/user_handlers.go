package api

import (
	"context"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// TODO(SV-61): When user registration, password change, or admin user-management
// endpoints are added, call pwdcheck.Validate(ctx, password, h.PasswordChecker)
// before hashing. When a login endpoint detects a breached password, require a
// password reset.

func (h *Handler) UsersMe(ctx context.Context) (*apigen.User, error) {
	authUser := auth.UserFromContext(ctx)
	q := dbgen.New(h.Pool)
	user, err := q.GetUserByID(ctx, authUser.ID)
	if err != nil {
		return nil, err
	}
	return userFromDB(user), nil
}
