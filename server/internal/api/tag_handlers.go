package api

import (
	"context"
	"time"

	"github.com/sargunv/tend/server/internal/activitylog"
	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) SpaceTagsList(ctx context.Context, params apigen.SpaceTagsListParams) (*apigen.TagList, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)

	rows, err := q.ListAllTagsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	return &apigen.TagList{Items: convertAll(rows, tagFromDB)}, nil
}

func (h *Handler) SpaceTagsCreate(ctx context.Context, req *apigen.TagCreate, params apigen.SpaceTagsCreateParams) (*apigen.Tag, error) {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	name := req.Name
	if err := validateTagName(name); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	now := time.Now()
	tag, err := q.CreateTag(ctx, dbgen.CreateTagParams{
		SpaceSlug:  params.SpaceSlug,
		Name:       name,
		NameFolded: taskengine.FoldTagName(name),
		CreatedAt:  types.Timestamptz(now),
	})
	if err != nil {
		return nil, err
	}

	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityTag,
		EntityID:   name,
		Action:     activitylog.ActionCreated,
	}, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return tagFromDB(tag), nil
}

func (h *Handler) SpaceTagsUpdate(ctx context.Context, req *apigen.TagUpdate, params apigen.SpaceTagsUpdateParams) (*apigen.Tag, error) {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	newName := req.Name
	if err := validateTagName(newName); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetTagByFoldedName(ctx, dbgen.GetTagByFoldedNameParams{
		SpaceSlug:  params.SpaceSlug,
		NameFolded: taskengine.FoldTagName(params.TagName),
	})
	if err != nil {
		return nil, err
	}

	tag, err := q.UpdateTag(ctx, dbgen.UpdateTagParams{
		Name:       newName,
		NameFolded: taskengine.FoldTagName(newName),
		ID:         existing.ID,
		SpaceSlug:  params.SpaceSlug,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityTag,
		EntityID:   newName,
		Action:     activitylog.ActionUpdated,
		Details: []activitylog.Detail{
			{Field: "name", From: new(existing.Name), To: new(newName)},
		},
	}, now); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return tagFromDB(tag), nil
}

func (h *Handler) SpaceTagsDelete(ctx context.Context, params apigen.SpaceTagsDeleteParams) error {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetTagByFoldedName(ctx, dbgen.GetTagByFoldedNameParams{
		SpaceSlug:  params.SpaceSlug,
		NameFolded: taskengine.FoldTagName(params.TagName),
	})
	if err != nil {
		return err
	}

	result, err := q.DeleteTag(ctx, dbgen.DeleteTagParams{
		SpaceSlug:  params.SpaceSlug,
		NameFolded: taskengine.FoldTagName(params.TagName),
	})
	if err != nil {
		return err
	}
	if err := checkDeleted(result); err != nil {
		return err
	}

	now := time.Now()
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityTag,
		EntityID:   existing.Name,
		Action:     activitylog.ActionDeleted,
	}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
