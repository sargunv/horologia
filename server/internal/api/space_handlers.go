package api

import (
	"context"
	"time"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// --- Spaces ---

func (h *Handler) SpacesCreate(ctx context.Context, req *apigen.SpaceCreate) (*apigen.Space, error) {
	user := UserFromContext(ctx)
	space, err := taskengine.CreateSpaceWithDefaults(
		ctx, h.Pool,
		req.Slug,
		req.Name,
		req.Description.Or(""),
		user.ID,
	)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space), nil
}

func (h *Handler) SpacesList(ctx context.Context) (*apigen.SpaceList, error) {
	user := UserFromContext(ctx)

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

	space, err := q.UpdateSpace(ctx, dbgen.UpdateSpaceParams{
		Slug:        req.Slug.Or(existing.Slug),
		Name:        req.Name.Or(existing.Name),
		Description: req.Description.Or(existing.Description),
		UpdatedAt:   types.Timestamptz(time.Now()),
		Slug_2:      params.SpaceSlug,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return spaceFromDB(space), nil
}

func (h *Handler) SpacesDelete(ctx context.Context, params apigen.SpacesDeleteParams) error {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return err
	}
	q := dbgen.New(h.Pool)
	result, err := q.DeleteSpace(ctx, params.SpaceSlug)
	if err != nil {
		return err
	}
	return checkDeleted(result)
}
