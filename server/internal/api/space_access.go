package api

import (
	"context"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// requireSpaceRole checks that the authenticated user has one of the given roles
// in the specified space. Global owners always pass.
func (h *Handler) requireSpaceRole(ctx context.Context, spaceSlug string, roles ...string) error {
	user := UserFromContext(ctx)
	if user.IsOwner {
		return nil
	}
	q := dbgen.New(h.DB)
	member, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: spaceSlug,
		UserID:    user.ID,
	})
	if err != nil {
		return err // sql.ErrNoRows -> 404 via NewError
	}
	for _, r := range roles {
		if member.Role == r {
			return nil
		}
	}
	return badRequest("insufficient permissions")
}

// requireSpaceWrite checks that the user has member or admin role.
func (h *Handler) requireSpaceWrite(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "member", "admin")
}

// requireSpaceRead checks that the user has any role in the space.
func (h *Handler) requireSpaceRead(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "viewer", "member", "admin")
}
