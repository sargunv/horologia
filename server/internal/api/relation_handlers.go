package api

import (
	"context"
	"time"

	"github.com/sargunv/tend/server/internal/activitylog"
	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
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

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

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
		CreatedAt:    types.Timestamptz(ts),
	}); err != nil {
		return nil, err
	}

	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  params.SpaceSlug,
		EntityType: activitylog.EntityRelation,
		EntityID:   formatTaskID(sourceID),
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "kind", To: new(string(req.Kind))},
			{Field: "related_task", To: new(formatTaskID(targetID))},
		},
	}, ts); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
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

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	result, err := q.DeleteTaskRelation(ctx, dbgen.DeleteTaskRelationParams{
		SourceTaskID: storedSource,
		TargetTaskID: storedTarget,
		Kind:         storedKind,
		SpaceSlug:    params.SpaceSlug,
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
		EntityType: activitylog.EntityRelation,
		EntityID:   formatTaskID(sourceID),
		Action:     activitylog.ActionDeleted,
		Details: []activitylog.Detail{
			{Field: "kind", From: new(string(params.Kind))},
			{Field: "related_task", From: new(formatTaskID(targetID))},
		},
	}, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
