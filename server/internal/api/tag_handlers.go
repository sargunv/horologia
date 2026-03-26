package api

import (
	"context"
	"strconv"
	"time"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) SpaceTagsList(ctx context.Context, params apigen.SpaceTagsListParams) (*apigen.TagPage, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)

	q := dbgen.New(h.Pool)

	rows, err := q.ListTagsBySpace(ctx, dbgen.ListTagsBySpaceParams{
		SpaceSlug: params.SpaceSlug,
		ID:        cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(rows, limit, convertEach(tagFromDB), tagCursorKey)
	if err != nil {
		return nil, err
	}

	return &apigen.TagPage{Items: items, NextCursor: nextCursor}, nil
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

	tag, err := q.CreateTag(ctx, dbgen.CreateTagParams{
		SpaceSlug:  params.SpaceSlug,
		Name:       name,
		NameFolded: taskengine.FoldTagName(name),
		CreatedAt:  types.Timestamptz(time.Now()),
	})
	if err != nil {
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return tagFromDB(tag), nil
}

func (h *Handler) SpaceTagsDelete(ctx context.Context, params apigen.SpaceTagsDeleteParams) error {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return err
	}

	q := dbgen.New(h.Pool)
	result, err := q.DeleteTag(ctx, dbgen.DeleteTagParams{
		SpaceSlug:  params.SpaceSlug,
		NameFolded: taskengine.FoldTagName(params.TagName),
	})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}

func tagCursorKey(t dbgen.Tag) string {
	return strconv.FormatInt(t.ID, 10)
}
