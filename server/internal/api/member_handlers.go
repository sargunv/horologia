package api

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) SpaceMembersList(ctx context.Context, params apigen.SpaceMembersListParams) (*apigen.SpaceMemberPage, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)

	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	rows, err := q.ListSpaceMembersBySpace(ctx, dbgen.ListSpaceMembersBySpaceParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(rows, limit, func(rows []dbgen.ListSpaceMembersBySpaceRow) ([]apigen.SpaceMember, error) {
		items := make([]apigen.SpaceMember, len(rows))
		for i, r := range rows {
			items[i] = *memberToAPI(r.UserID, r.UserName, r.UserEmail, r.Role, r.CreatedAt)
		}
		return items, nil
	}, func(r dbgen.ListSpaceMembersBySpaceRow) string {
		return strconv.FormatInt(r.UserID, 10)
	})
	if err != nil {
		return nil, err
	}

	return &apigen.SpaceMemberPage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) SpaceMembersCreate(ctx context.Context, req *apigen.SpaceMemberCreate, params apigen.SpaceMembersCreateParams) (*apigen.SpaceMember, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}

	userID, err := parseUserID(req.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	// Verify the target user exists.
	targetUser, err := q.GetUserByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, badRequest("user not found")
	}
	if err != nil {
		return nil, err
	}

	member, err := q.CreateSpaceMember(ctx, dbgen.CreateSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
		Role:      string(req.Role),
		CreatedAt: types.Now(),
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return memberToAPI(member.UserID, targetUser.Name, targetUser.Email, member.Role, member.CreatedAt), nil
}

func (h *Handler) SpaceMembersUpdate(ctx context.Context, req *apigen.SpaceMemberUpdate, params apigen.SpaceMembersUpdateParams) (*apigen.SpaceMember, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}

	userID, err := parseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	// Guard against removing the last admin.
	if req.Role != apigen.SpaceRoleAdmin {
		if err := h.ensureNotLastAdmin(ctx, q, params.SpaceSlug, userID); err != nil {
			return nil, err
		}
	}

	member, err := q.UpdateSpaceMemberRole(ctx, dbgen.UpdateSpaceMemberRoleParams{
		Role:      string(req.Role),
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}

	targetUser, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return memberToAPI(member.UserID, targetUser.Name, targetUser.Email, member.Role, member.CreatedAt), nil
}

func (h *Handler) SpaceMembersDelete(ctx context.Context, params apigen.SpaceMembersDeleteParams) error {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return err
	}

	userID, err := parseUserID(params.UserId)
	if err != nil {
		return badRequest(err.Error())
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	// Guard against removing the last admin.
	if err := h.ensureNotLastAdmin(ctx, q, params.SpaceSlug, userID); err != nil {
		return err
	}

	// Remove task assignments and rotation pool entries for this user in this space
	// before removing membership.
	if err := q.DeleteTaskAssigneesBySpaceAndUser(ctx, dbgen.DeleteTaskAssigneesBySpaceAndUserParams{
		UserID:    userID,
		SpaceSlug: params.SpaceSlug,
	}); err != nil {
		return err
	}
	if err := q.DeleteRotationPoolBySpaceAndUser(ctx, dbgen.DeleteRotationPoolBySpaceAndUserParams{
		UserID:    userID,
		SpaceSlug: params.SpaceSlug,
	}); err != nil {
		return err
	}

	result, err := q.DeleteSpaceMember(ctx, dbgen.DeleteSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if err := checkDeleted(result); err != nil {
		return err
	}

	return tx.Commit()
}

// ensureNotLastAdmin returns an error if the given user is the only admin in the space.
func (h *Handler) ensureNotLastAdmin(ctx context.Context, q *dbgen.Queries, spaceSlug string, userID int64) error {
	member, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: spaceSlug,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if member.Role != "admin" {
		return nil // not an admin, no risk
	}
	count, err := q.CountSpaceAdmins(ctx, spaceSlug)
	if err != nil {
		return err
	}
	if count <= 1 {
		return badRequest("cannot remove the last admin from a space")
	}
	return nil
}
