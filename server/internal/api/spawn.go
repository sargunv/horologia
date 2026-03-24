package api

import (
	"context"
	"time"

	"github.com/teambition/rrule-go"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// spawnTaskFromTemplate creates a new task by copying user-settable fields from src.
// The new task gets the provided recurrenceType, recurrenceRule, and dueAt.
// It copies assignees (or uses overrideAssignees if non-nil), tags, rotation pool
// (from srcPool), and copyOnSpawn=true relations from src.
// A "spawns" relation is created from src to the new task.
// Must be called within a transaction.
func (h *Handler) spawnTaskFromTemplate(
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
			NameFolded: foldTagName(name),
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

	// Copy relations where copyOnSpawn is true. Re-point: where src was
	// source, new becomes source; where src was target, new becomes target.
	asSource, err := q.ListRelationsByTaskAsSource(ctx, dbgen.ListRelationsByTaskAsSourceParams{
		SourceTaskID: src.ID,
		SpaceSlug:    src.SpaceSlug,
	})
	if err != nil {
		return 0, err
	}
	for _, r := range asSource {
		if !storedKindCopyOnSpawn[r.Kind] {
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
		if !storedKindCopyOnSpawn[r.Kind] {
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
// single fixed_accumulating task. This prevents unbounded task creation after
// extended server downtime. With FREQ=DAILY this covers ~1 year.
const maxMissedOccurrences = 365

// allOverdueOccurrences returns RRULE occurrences from dtstart up to and
// including `until`, capped at maxMissedOccurrences. Each occurrence is
// truncated to midnight in the given location. Returns nil if no occurrences
// exist in the range.
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

	// Get all occurrences from dtstart (exclusive — dtstart itself is the
	// current task's due date) through until (inclusive).
	occurrences := rr.Between(dtstart, until, false)

	// Cap to prevent unbounded spawning after extended downtime.
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
// existing transaction. For each missed occurrence it spawns a one_off task,
// then converts the original to one_off and creates a new fixed_accumulating
// task with the next future due date.
func (h *Handler) processAccumulatingTask(ctx context.Context, q *dbgen.Queries, task dbgen.Task, now time.Time) error {
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

	// Find all missed occurrences (strictly after dtstart, up to now).
	missed, err := allOverdueOccurrences(*task.RecurrenceRule, dtstart, now, loc)
	if err != nil {
		return err
	}

	initialStatus, err := findInitialStatus(ctx, q, task.SpaceSlug)
	if err != nil {
		return err
	}

	nowEpoch := types.EpochSecondsFrom(now)

	// Atomically convert the original task to one_off. The WHERE guard
	// (recurrence_type = 'fixed_accumulating') prevents double-processing
	// if the completion handler or another cron tick already converted it.
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
		return nil // already converted by another path
	}

	// Read rotation pool and current assignees for rotation across spawns.
	pool, err := q.ListRotationPoolByTask(ctx, task.ID)
	if err != nil {
		return err
	}
	currentAssignees, err := q.ListAssigneeUserIDsByTask(ctx, task.ID)
	if err != nil {
		return err
	}

	// Spawn one_off tasks for each missed occurrence, advancing rotation.
	for i, dueAt := range missed {
		overrideAssignees := advanceRotation(pool, currentAssignees, i)
		if _, err := h.spawnTaskFromTemplate(ctx, q, task,
			"one_off", nil, &dueAt, initialStatus, nowEpoch,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	// Compute the next future due date.
	next, err := computeNextDueAt("fixed_accumulating", task.RecurrenceRule, task.DueAt, task.DueTz, now)
	if err != nil {
		return err
	}

	if next != nil {
		// Spawn a new fixed_accumulating task with the next future due date.
		// Rotation advances past all missed occurrences.
		overrideAssignees := advanceRotation(pool, currentAssignees, len(missed))
		if _, err := h.spawnTaskFromTemplate(ctx, q, task,
			"fixed_accumulating", task.RecurrenceRule, next, initialStatus, nowEpoch,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	// Clear the rotation pool from the now-one_off original task.
	if err := q.DeleteRotationPool(ctx, task.ID); err != nil {
		return err
	}

	return nil
}
