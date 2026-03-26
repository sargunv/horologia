package api

import (
	"context"
	"time"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// rejectSpawnKind returns a validation error if the given kind is a
// system-managed spawn relation that users cannot create or delete.
func rejectSpawnKind(kind apigen.TaskRelationKind) error {
	if kind == apigen.TaskRelationKindSpawns || kind == apigen.TaskRelationKindSpawnedBy {
		return badRequest("spawns/spawned_by relations are system-managed and cannot be created or deleted manually")
	}
	return nil
}

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

	if err := rejectSpawnKind(req.Kind); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)

	// Verify target task exists and is in the same space.
	_, err = q.GetTask(ctx, dbgen.GetTaskParams{ID: targetID, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return nil, err
	}

	storedKind, storedSource, storedTarget, err := canonicalizeRelation(req.Kind, sourceID, targetID)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	ts := time.Now()
	if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
		SourceTaskID: storedSource,
		TargetTaskID: storedTarget,
		SpaceSlug:    params.SpaceSlug,
		Kind:         storedKind,
		CreatedAt:    timeToTS(ts),
	}); err != nil {
		return nil, err
	}

	return &apigen.TaskRelation{
		Kind:      req.Kind,
		TaskId:    formatTaskID(targetID),
		CreatedAt: ts,
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

	if err := rejectSpawnKind(params.Kind); err != nil {
		return err
	}

	storedKind, storedSource, storedTarget, err := canonicalizeRelation(params.Kind, sourceID, targetID)
	if err != nil {
		return badRequest(err.Error())
	}

	q := dbgen.New(h.Pool)
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
