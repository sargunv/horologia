package api

import (
	"context"
	"slices"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// requireSpaceRole checks that the authenticated user has one of the given roles
// in the specified space. Global owners always pass, but the space must still
// exist to prevent a 200 OK for a nonexistent space.
func (h *Handler) requireSpaceRole(ctx context.Context, spaceSlug string, roles ...dbgen.SpaceRole) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return &ogenerrors.SecurityError{Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied}
	}
	q := dbgen.New(h.Pool)
	if user.IsOwner {
		if _, err := q.GetSpace(ctx, spaceSlug); err != nil {
			return err
		}
		return nil
	}
	member, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: spaceSlug,
		UserID:    user.ID,
	})
	if err != nil {
		return err // pgx.ErrNoRows -> 404 via NewError
	}
	if slices.Contains(roles, member.Role) {
		return nil
	}
	return forbidden("insufficient permissions")
}

// requireSpaceWrite checks that the user has member or admin role.
func (h *Handler) requireSpaceWrite(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, dbgen.SpaceRoleMember, dbgen.SpaceRoleAdmin)
}

// requireSpaceRead checks that the user has any role in the space.
func (h *Handler) requireSpaceRead(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, dbgen.SpaceRoleViewer, dbgen.SpaceRoleMember, dbgen.SpaceRoleAdmin)
}
