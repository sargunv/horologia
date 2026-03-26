package api

import (
	"context"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

func (h *Handler) UsersMe(ctx context.Context) (*apigen.User, error) {
	authUser := auth.UserFromContext(ctx)
	q := dbgen.New(h.Pool)
	user, err := q.GetUserByID(ctx, authUser.ID)
	if err != nil {
		return nil, err
	}
	return userFromDB(user), nil
}
