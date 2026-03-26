package api

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/sargunv/tend/server/internal/activitylog"
	apigen "github.com/sargunv/tend/server/internal/api/gen"
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

	q := dbgen.New(h.Pool)

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
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	userID, err := types.ParseUserID(req.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Verify the target user exists.
	targetUser, err := q.GetUserByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, badRequest("user not found")
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	member, err := q.CreateSpaceMember(ctx, dbgen.CreateSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
		Role:      dbgen.SpaceRole(req.Role),
		CreatedAt: types.Timestamptz(now),
	})
	if err != nil {
		return nil, err
	}

	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityMember,
		EntityID:   types.FormatUserID(userID),
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "role", To: new(string(req.Role))},
		},
	}, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return memberToAPI(member.UserID, targetUser.Name, targetUser.Email, member.Role, member.CreatedAt), nil
}

func (h *Handler) SpaceMembersUpdate(ctx context.Context, req *apigen.SpaceMemberUpdate, params apigen.SpaceMembersUpdateParams) (*apigen.SpaceMember, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	userID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Capture existing role before the update.
	existingMember, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}

	// Atomically updates the role; refuses to demote the last admin (returns no rows).
	member, err := q.UpdateSpaceMemberRole(ctx, dbgen.UpdateSpaceMemberRoleParams{
		Role:      dbgen.SpaceRole(req.Role),
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, badRequest("cannot remove the last admin from a space")
	}
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if string(existingMember.Role) != string(req.Role) {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityMember,
			EntityID:   types.FormatUserID(userID),
			Action:     activitylog.ActionUpdated,
			Details: []activitylog.Detail{
				{Field: "role", From: new(string(existingMember.Role)), To: new(string(req.Role))},
			},
		}, now); err != nil {
			return nil, err
		}
	}

	targetUser, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return memberToAPI(member.UserID, targetUser.Name, targetUser.Email, member.Role, member.CreatedAt), nil
}

func (h *Handler) SpaceMembersDelete(ctx context.Context, params apigen.SpaceMembersDeleteParams) error {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return err
	}

	userID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return badRequest(err.Error())
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Capture existing role before deletion for the log entry.
	existingMember, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if err != nil {
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

	// Atomically deletes the member; refuses to delete the last admin (affects zero rows).
	result, err := q.DeleteSpaceMember(ctx, dbgen.DeleteSpaceMemberParams{
		SpaceSlug: params.SpaceSlug,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		if existingMember.Role == dbgen.SpaceRoleAdmin {
			return badRequest("cannot remove the last admin from a space")
		}
		return pgx.ErrNoRows
	}

	now := time.Now()
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityMember,
		EntityID:   types.FormatUserID(userID),
		Action:     activitylog.ActionDeleted,
		Details: []activitylog.Detail{
			{Field: "role", From: new(string(existingMember.Role))},
		},
	}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
