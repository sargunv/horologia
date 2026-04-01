package api

import (
	"context"
	"strconv"
	"time"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/sargunv/tend/server/internal/activitylog"
	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) UsersList(ctx context.Context) (*apigen.UserList, error) {
	q := dbgen.New(h.Pool)
	users, err := q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &apigen.UserList{Items: convertAll(users, userFromDB)}, nil
}

func (h *Handler) UsersGet(ctx context.Context, params apigen.UsersGetParams) (*apigen.User, error) {
	userID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	q := dbgen.New(h.Pool)
	user, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userFromDB(user), nil
}

func (h *Handler) UsersCreate(ctx context.Context, req *apigen.UserCreate) (*apigen.User, error) {
	if err := h.requireOwner(ctx); err != nil {
		return nil, err
	}

	isOwner := req.IsOwner.Or(false)

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()

	var user dbgen.User
	if req.Password.IsSet() {
		user, err = taskengine.CreateUserWithPassword(ctx, tx, req.Email, req.Name, req.Password.Value, isOwner, h.PasswordChecker, now)
	} else {
		user, err = taskengine.CreateUserWithoutPassword(ctx, tx, req.Email, req.Name, isOwner, now)
	}
	if err != nil {
		return nil, err
	}
	ownerStr := strconv.FormatBool(user.IsOwner)
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  "",
		EntityType: activitylog.EntityUser,
		EntityID:   types.FormatUserID(user.ID),
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "email", To: new(user.Email)},
			{Field: "name", To: new(user.Name)},
			{Field: "isOwner", To: &ownerStr},
		},
	}, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return userFromDB(user), nil
}

func (h *Handler) UsersUpdate(ctx context.Context, req *apigen.UserUpdate, params apigen.UsersUpdateParams) (*apigen.User, error) {
	authUser := auth.UserFromContext(ctx)
	if authUser == nil {
		return nil, &ogenerrors.SecurityError{Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied}
	}

	userID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	// Non-owners can only update themselves.
	if !authUser.IsOwner {
		if userID != authUser.ID {
			return nil, forbidden("cannot modify other users")
		}
		// Non-owners cannot change owner status.
		if req.IsOwner.IsSet() {
			return nil, forbidden("cannot change owner status")
		}
	}

	if req.SetPassword.IsSet() && req.ClearPassword.Or(false) {
		return nil, badRequest("cannot set and clear password simultaneously")
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	newName := req.Name.Or(existing.Name)
	newEmail := req.Email.Or(existing.Email)
	newIsOwner := req.IsOwner.Or(existing.IsOwner)

	// Prevent demoting the last owner.
	if existing.IsOwner && !newIsOwner {
		count, err := q.CountOwners(ctx)
		if err != nil {
			return nil, err
		}
		if count <= 1 {
			return nil, badRequest("cannot demote the last owner")
		}
	}

	now := time.Now()
	updated, err := q.UpdateUser(ctx, dbgen.UpdateUserParams{
		Name:      newName,
		Email:     newEmail,
		IsOwner:   newIsOwner,
		UpdatedAt: types.Timestamptz(now),
		ID:        userID,
	})
	if err != nil {
		return nil, err
	}

	// Handle password changes.
	if req.SetPassword.IsSet() {
		if err := taskengine.SetUserPassword(ctx, tx, userID, req.SetPassword.Value, h.PasswordChecker, now); err != nil {
			return nil, err
		}
		// Revoke all other session tokens for this user so stale sessions
		// cannot survive a password change. API tokens are not affected.
		if err := q.DeleteOtherSessionTokens(ctx, dbgen.DeleteOtherSessionTokensParams{
			UserID:    userID,
			TokenHash: authUser.SessionTokenHash,
		}); err != nil {
			return nil, err
		}
	} else if req.ClearPassword.Or(false) {
		if err := taskengine.ClearUserPassword(ctx, tx, userID, now); err != nil {
			return nil, err
		}
	}

	// Log changed fields.
	var details []activitylog.Detail
	if newName != existing.Name {
		details = append(details, activitylog.Detail{Field: "name", From: new(existing.Name), To: new(newName)})
	}
	if newEmail != existing.Email {
		details = append(details, activitylog.Detail{Field: "email", From: new(existing.Email), To: new(newEmail)})
	}
	if newIsOwner != existing.IsOwner {
		fromOwner := strconv.FormatBool(existing.IsOwner)
		toOwner := strconv.FormatBool(newIsOwner)
		details = append(details, activitylog.Detail{Field: "isOwner", From: &fromOwner, To: &toOwner})
	}
	if req.SetPassword.IsSet() {
		set := "(set)"
		details = append(details, activitylog.Detail{Field: "password", To: &set})
	}
	if req.ClearPassword.Or(false) {
		cleared := "(cleared)"
		d := activitylog.Detail{Field: "password", To: &cleared}
		if existing.PasswordHash.Valid {
			was := "(set)"
			d.From = &was
		}
		details = append(details, d)
	}
	if len(details) > 0 {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  "",
			EntityType: activitylog.EntityUser,
			EntityID:   types.FormatUserID(userID),
			Action:     activitylog.ActionUpdated,
			Details:    details,
		}, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return userFromDB(updated), nil
}

func (h *Handler) UsersDelete(ctx context.Context, params apigen.UsersDeleteParams) error {
	authUser := auth.UserFromContext(ctx)
	if authUser == nil {
		return &ogenerrors.SecurityError{Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied}
	}

	userID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return badRequest(err.Error())
	}

	// Non-owners can only delete themselves.
	if !authUser.IsOwner && userID != authUser.ID {
		return forbidden("cannot modify other users")
	}

	now := time.Now()

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	// Prevent deleting the last owner.
	if existing.IsOwner {
		count, err := q.CountOwners(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return badRequest("cannot delete the last owner")
		}
	}

	result, err := q.DeleteUser(ctx, userID)
	if err != nil {
		return err
	}
	if err := checkDeleted(result); err != nil {
		return err
	}

	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  "",
		EntityType: activitylog.EntityUser,
		EntityID:   types.FormatUserID(userID),
		Action:     activitylog.ActionDeleted,
		Details: []activitylog.Detail{
			{Field: "email", From: new(existing.Email)},
			{Field: "name", From: new(existing.Name)},
		},
	}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
