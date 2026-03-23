package api

import (
	"context"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) SpaceTaskRelationsCreate(ctx context.Context, req *apigen.TaskRelationCreate, params apigen.SpaceTaskRelationsCreateParams) (*apigen.TaskRelation, error) {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	sourceID, err := parseTaskID(params.TaskId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	targetID, err := parseTaskID(req.TaskId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	if sourceID == targetID {
		return nil, badRequest("a task cannot be related to itself")
	}

	q := dbgen.New(h.DB)

	// Verify target task exists and is in the same space.
	targetTask, err := q.GetTask(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if targetTask.SpaceSlug != params.SpaceSlug {
		return nil, badRequest("tasks must be in the same space")
	}

	storedKind, storedSource, storedTarget := canonicalizeRelation(req.Kind, sourceID, targetID)

	ts := types.Now()
	if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
		SourceTaskID: storedSource,
		TargetTaskID: storedTarget,
		SpaceSlug:    params.SpaceSlug,
		Kind:         storedKind,
		CreatedAt:    ts,
	}); err != nil {
		return nil, err
	}

	return &apigen.TaskRelation{
		Kind:      req.Kind,
		TaskId:    formatTaskID(targetID),
		CreatedAt: ts.Time(),
	}, nil
}

func (h *Handler) SpaceTaskRelationsDelete(ctx context.Context, params apigen.SpaceTaskRelationsDeleteParams) error {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return err
	}

	sourceID, err := parseTaskID(params.TaskId)
	if err != nil {
		return badRequest(err.Error())
	}
	targetID, err := parseTaskID(params.RelatedTaskId)
	if err != nil {
		return badRequest(err.Error())
	}

	storedKind, storedSource, storedTarget := canonicalizeRelation(params.Kind, sourceID, targetID)

	q := dbgen.New(h.DB)
	result, err := q.DeleteTaskRelation(ctx, dbgen.DeleteTaskRelationParams{
		SourceTaskID: storedSource,
		TargetTaskID: storedTarget,
		Kind:         storedKind,
	})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}
