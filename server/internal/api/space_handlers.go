package api

import (
	"context"
	"time"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/sargunv/tend/server/internal/activitylog"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// --- Spaces ---

func (h *Handler) SpacesCreate(ctx context.Context, req *apigen.SpaceCreate) (*apigen.Space, error) {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return nil, err
	}
	user := auth.UserFromContext(ctx)
	space, err := taskengine.CreateSpaceWithDefaults(
		ctx, h.Pool,
		req.Slug,
		req.Name,
		req.Description.Or(""),
		user.ID,
		time.Now(),
	)
	if err != nil {
		return nil, err
	}

	return spaceFromDB(space), nil
}

func (h *Handler) SpacesList(ctx context.Context) (*apigen.SpaceList, error) {
	if err := h.requireScope(ctx, "spaces:read"); err != nil {
		return nil, err
	}
	user := auth.UserFromContext(ctx)

	q := dbgen.New(h.Pool)

	var (
		spaces []dbgen.Space
		err    error
	)
	if user.IsOwner {
		spaces, err = q.ListSpaces(ctx)
	} else {
		spaces, err = q.ListSpacesByUser(ctx, user.ID)
	}
	if err != nil {
		return nil, err
	}

	return &apigen.SpaceList{Items: convertAll(spaces, spaceFromDB)}, nil
}

func (h *Handler) SpacesRead(ctx context.Context, params apigen.SpacesReadParams) (*apigen.Space, error) {
	if err := h.requireScope(ctx, "spaces:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	q := dbgen.New(h.Pool)
	space, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space), nil
}

func (h *Handler) SpacesUpdate(ctx context.Context, req *apigen.SpaceUpdate, params apigen.SpacesUpdateParams) (*apigen.Space, error) {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newSlug := req.Slug.Or(existing.Slug)
	newName := req.Name.Or(existing.Name)
	newDescription := req.Description.Or(existing.Description)

	space, err := q.UpdateSpace(ctx, dbgen.UpdateSpaceParams{
		Slug:        newSlug,
		Name:        newName,
		Description: newDescription,
		UpdatedAt:   types.Timestamptz(now),
		Slug_2:      params.SpaceSlug,
	})
	if err != nil {
		return nil, err
	}

	var details []activitylog.Detail
	if newSlug != existing.Slug {
		details = append(details, activitylog.Detail{Field: "slug", From: new(existing.Slug), To: new(newSlug)})
	}
	if newName != existing.Name {
		details = append(details, activitylog.Detail{Field: "name", From: new(existing.Name), To: new(newName)})
	}
	if newDescription != existing.Description {
		details = append(details, activitylog.Detail{Field: "description", From: new(existing.Description), To: new(newDescription)})
	}
	if len(details) > 0 {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  space.Slug,
			EntityType: activitylog.EntitySpace,
			EntityID:   space.Slug,
			Action:     activitylog.ActionUpdated,
			Details:    details,
		}, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return spaceFromDB(space), nil
}

func (h *Handler) SpacesDelete(ctx context.Context, params apigen.SpacesDeleteParams) error {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return err
	}
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return err
	}

	result, err := q.DeleteSpace(ctx, params.SpaceSlug)
	if err != nil {
		return err
	}
	if err := checkDeleted(result); err != nil {
		return err
	}

	now := time.Now()
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntitySpace,
		EntityID:   params.SpaceSlug,
		Action:     activitylog.ActionDeleted,
		Details: []activitylog.Detail{
			{Field: "name", From: new(existing.Name)},
		},
	}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
