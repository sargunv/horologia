package api

import (
	"context"
	"slices"

	"github.com/ogen-go/ogen/ogenerrors"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// requireSpaceRole checks that the authenticated user has one of the given roles
// in the specified space. Global owners always pass but the space must exist
// (prevents returning empty 200 for nonexistent spaces).
func (h *Handler) requireSpaceRole(ctx context.Context, spaceSlug string, roles ...string) error {
	user := UserFromContext(ctx)
	if user == nil {
		return &ogenerrors.SecurityError{Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied}
	}
	if user.IsOwner {
		q := dbgen.New(h.DB)
		if _, err := q.GetSpace(ctx, spaceSlug); err != nil {
			return err
		}
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
	if slices.Contains(roles, member.Role) {
		return nil
	}
	return forbidden("insufficient permissions")
}

// requireSpaceWrite checks that the user has member or admin role.
func (h *Handler) requireSpaceWrite(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "member", "admin")
}

// requireSpaceRead checks that the user has any role in the space.
func (h *Handler) requireSpaceRead(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "viewer", "member", "admin")
}
