package taskengine

import (
	"context"
	"time"

	"github.com/teambition/rrule-go"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// SpawnTaskFromTemplate creates a new task by copying user-settable fields from src.
// The new task gets the provided recurrenceType, recurrenceRule, and dueAt.
// It copies assignees (or uses overrideAssignees if non-nil), tags, rotation pool
// (from srcPool), and copyOnSpawn=true relations from src.
// A "spawns" relation is created from src to the new task.
// Must be called within a transaction.
func (e *Engine) SpawnTaskFromTemplate(
	ctx context.Context,
	q *dbgen.Queries,
	src dbgen.Task,
	newRecurrenceType string,
	newRecurrenceRule *string,
	newDueAt *types.EpochSeconds,
	initialStatus string,
	now types.EpochSeconds,
	overrideAssignees []int64,
	srcPool []int64,
) (int64, error) {
	newTask, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		SpaceSlug:      src.SpaceSlug,
		Title:          src.Title,
		Description:    src.Description,
		StatusName:     initialStatus,
		EffortName:     src.EffortName,
		PriorityName:   src.PriorityName,
		DueAt:          newDueAt,
		DueTz:          src.DueTz,
		RecurrenceType: newRecurrenceType,
		RecurrenceRule: newRecurrenceRule,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return 0, err
	}

	// Copy assignees, or use overrideAssignees if provided (for rotation).
	assigneeIDs := overrideAssignees
	if assigneeIDs == nil {
		assigneeIDs, err = q.ListAssigneeUserIDsByTask(ctx, src.ID)
		if err != nil {
			return 0, err
		}
	}
	for _, uid := range assigneeIDs {
		if err := q.InsertTaskAssignee(ctx, dbgen.InsertTaskAssigneeParams{
			TaskID:    newTask.ID,
			UserID:    uid,
			CreatedAt: now,
		}); err != nil {
			return 0, err
		}
	}

	// Copy tags.
	tagNames, err := q.ListTagNamesByTask(ctx, src.ID)
	if err != nil {
		return 0, err
	}
	for _, name := range tagNames {
		tag, err := q.EnsureTag(ctx, dbgen.EnsureTagParams{
			SpaceSlug:  src.SpaceSlug,
			Name:       name,
			NameFolded: FoldTagName(name),
			CreatedAt:  now,
		})
		if err != nil {
			return 0, err
		}
		if err := q.InsertTaskTag(ctx, dbgen.InsertTaskTagParams{
			TaskID:    newTask.ID,
			TagID:     tag.ID,
			CreatedAt: now,
		}); err != nil {
			return 0, err
		}
	}

	// Copy rotation pool from pre-fetched srcPool.
	for i, uid := range srcPool {
		if err := q.InsertRotationPoolMember(ctx, dbgen.InsertRotationPoolMemberParams{
			TaskID:    newTask.ID,
			UserID:    uid,
			Position:  int64(i),
			CreatedAt: now,
		}); err != nil {
			return 0, err
		}
	}

	// Copy relations where copyOnSpawn is true.
	asSource, err := q.ListRelationsByTaskAsSource(ctx, dbgen.ListRelationsByTaskAsSourceParams{
		SourceTaskID: src.ID,
		SpaceSlug:    src.SpaceSlug,
	})
	if err != nil {
		return 0, err
	}
	for _, r := range asSource {
		if !e.CopyOnSpawnKinds[r.Kind] {
			continue
		}
		if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
			SourceTaskID: newTask.ID,
			TargetTaskID: r.TargetTaskID,
			SpaceSlug:    src.SpaceSlug,
			Kind:         r.Kind,
			CreatedAt:    now,
		}); err != nil {
			return 0, err
		}
	}

	asTarget, err := q.ListRelationsByTaskAsTarget(ctx, dbgen.ListRelationsByTaskAsTargetParams{
		TargetTaskID: src.ID,
		SpaceSlug:    src.SpaceSlug,
	})
	if err != nil {
		return 0, err
	}
	for _, r := range asTarget {
		if !e.CopyOnSpawnKinds[r.Kind] {
			continue
		}
		if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
			SourceTaskID: r.SourceTaskID,
			TargetTaskID: newTask.ID,
			SpaceSlug:    src.SpaceSlug,
			Kind:         r.Kind,
			CreatedAt:    now,
		}); err != nil {
			return 0, err
		}
	}

	// Create spawns relation: src → newTask.
	if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
		SourceTaskID: src.ID,
		TargetTaskID: newTask.ID,
		SpaceSlug:    src.SpaceSlug,
		Kind:         "spawns",
		CreatedAt:    now,
	}); err != nil {
		return 0, err
	}

	return newTask.ID, nil
}

// maxMissedOccurrences caps the number of tasks spawned per cron tick for a
// single fixed_accumulating task.
const maxMissedOccurrences = 365

// allOverdueOccurrences returns RRULE occurrences from dtstart up to and
// including `until`, capped at maxMissedOccurrences.
func allOverdueOccurrences(rule string, dtstart time.Time, until time.Time, loc *time.Location) ([]types.EpochSeconds, error) {
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return nil, err
	}
	opt.Dtstart = dtstart

	rr, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, err
	}

	occurrences := rr.Between(dtstart, until, false)

	if len(occurrences) > maxMissedOccurrences {
		occurrences = occurrences[len(occurrences)-maxMissedOccurrences:]
	}

	result := make([]types.EpochSeconds, 0, len(occurrences))
	for _, occ := range occurrences {
		midnight := time.Date(occ.Year(), occ.Month(), occ.Day(), 0, 0, 0, 0, loc)
		result = append(result, types.EpochSecondsFrom(midnight))
	}
	return result, nil
}

// processAccumulatingTask handles one overdue fixed_accumulating task within an
// existing transaction.
func (e *Engine) processAccumulatingTask(ctx context.Context, q *dbgen.Queries, task dbgen.Task, now time.Time) error {
	if task.DueAt == nil || task.RecurrenceRule == nil {
		return nil
	}

	loc := time.UTC
	if task.DueTz != nil {
		var err error
		loc, err = time.LoadLocation(*task.DueTz)
		if err != nil {
			return err
		}
	}

	dtstart := task.DueAt.Time().In(loc)

	missed, err := allOverdueOccurrences(*task.RecurrenceRule, dtstart, now, loc)
	if err != nil {
		return err
	}

	initialStatus, err := FindInitialStatus(ctx, q, task.SpaceSlug)
	if err != nil {
		return err
	}

	nowEpoch := types.EpochSecondsFrom(now)

	result, err := q.ConvertAccumulatingToOneOff(ctx, dbgen.ConvertAccumulatingToOneOffParams{
		UpdatedAt: nowEpoch,
		ID:        task.ID,
		SpaceSlug: task.SpaceSlug,
	})
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return nil
	}

	pool, err := q.ListRotationPoolByTask(ctx, task.ID)
	if err != nil {
		return err
	}
	currentAssignees, err := q.ListAssigneeUserIDsByTask(ctx, task.ID)
	if err != nil {
		return err
	}

	for i, dueAt := range missed {
		overrideAssignees := AdvanceRotation(pool, currentAssignees, i)
		if _, err := e.SpawnTaskFromTemplate(ctx, q, task,
			string(apigen.TaskRecurrenceTypeOneOff), nil, &dueAt, initialStatus, nowEpoch,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	next, err := ComputeNextDueAt(apigen.TaskRecurrenceTypeFixedAccumulating, task.RecurrenceRule, task.DueAt, task.DueTz, now)
	if err != nil {
		return err
	}

	if next != nil {
		overrideAssignees := AdvanceRotation(pool, currentAssignees, len(missed))
		if _, err := e.SpawnTaskFromTemplate(ctx, q, task,
			string(apigen.TaskRecurrenceTypeFixedAccumulating), task.RecurrenceRule, next, initialStatus, nowEpoch,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	if err := q.DeleteRotationPool(ctx, task.ID); err != nil {
		return err
	}

	return nil
}
