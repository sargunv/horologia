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

	// spawns/spawned_by are system-managed relations created by the recurrence
	// engine; users cannot create or manipulate them directly.
	if req.Kind == apigen.TaskRelationKindSpawns || req.Kind == apigen.TaskRelationKindSpawnedBy {
		return nil, badRequest("spawns/spawned_by relations are system-managed and cannot be created manually")
	}

	q := dbgen.New(h.DB)

	// Verify target task exists and is in the same space.
	_, err = q.GetTask(ctx, dbgen.GetTaskParams{ID: targetID, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return nil, err
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

	if params.Kind == apigen.TaskRelationKindSpawns || params.Kind == apigen.TaskRelationKindSpawnedBy {
		return badRequest("spawns/spawned_by relations are system-managed and cannot be deleted manually")
	}

	storedKind, storedSource, storedTarget := canonicalizeRelation(params.Kind, sourceID, targetID)

	q := dbgen.New(h.DB)
	result, err := q.DeleteTaskRelation(ctx, dbgen.DeleteTaskRelationParams{
		SourceTaskID: storedSource,
		TargetTaskID: storedTarget,
		Kind:         storedKind,
		SpaceSlug:    params.SpaceSlug,
	})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}
