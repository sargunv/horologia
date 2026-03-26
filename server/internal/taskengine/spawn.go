package taskengine

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/teambition/rrule-go"

	"github.com/sargunv/tend/server/internal/activitylog"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// copyOnSpawn returns true for relation kinds that should be copied when spawning
// a new task from a fixed_accumulating template.
func copyOnSpawn(k dbgen.StoredRelationKind) bool {
	switch k {
	case dbgen.StoredRelationKindParent, dbgen.StoredRelationKindBlocks, dbgen.StoredRelationKindTriggers, dbgen.StoredRelationKindRelatesTo:
		return true
	case dbgen.StoredRelationKindSpawns, dbgen.StoredRelationKindDuplicates:
		return false
	}
	return false
}

// SpawnTaskFromTemplate creates a new task by copying user-settable fields from src.
// The new task gets the provided recurrenceType, recurrenceRule, and dueAt.
// It copies assignees (or uses overrideAssignees if non-nil), tags, rotation pool
// (from srcPool), and copyOnSpawn=true relations from src.
// A "spawns" relation is created from src to the new task.
func SpawnTaskFromTemplate(
	ctx context.Context,
	db database.DB,
	src dbgen.Task,
	newRecurrenceType dbgen.RecurrenceType,
	newRecurrenceRule pgtype.Text,
	newDue *types.DueDate,
	initialStatus string,
	now time.Time,
	overrideAssignees []int64,
	srcPool []int64,
) (int64, error) {
	q := dbgen.New(db)
	dueAt, dueTz := types.DecomposeDueDate(newDue)
	nowTz := types.Timestamptz(now)
	newTask, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		SpaceSlug:      src.SpaceSlug,
		Title:          src.Title,
		Description:    src.Description,
		StatusName:     initialStatus,
		EffortName:     src.EffortName,
		PriorityName:   src.PriorityName,
		DueAt:          dueAt,
		DueTz:          dueTz,
		RecurrenceType: newRecurrenceType,
		RecurrenceRule: newRecurrenceRule,
		CreatedAt:      nowTz,
		UpdatedAt:      nowTz,
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
			CreatedAt: nowTz,
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
			CreatedAt:  nowTz,
		})
		if err != nil {
			return 0, err
		}
		if err := q.InsertTaskTag(ctx, dbgen.InsertTaskTagParams{
			TaskID:    newTask.ID,
			TagID:     tag.ID,
			CreatedAt: nowTz,
		}); err != nil {
			return 0, err
		}
	}

	// Copy rotation pool from pre-fetched srcPool.
	for i, uid := range srcPool {
		if err := q.InsertRotationPoolMember(ctx, dbgen.InsertRotationPoolMemberParams{
			TaskID:    newTask.ID,
			UserID:    uid,
			Position:  int32(i),
			CreatedAt: nowTz,
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
		if !copyOnSpawn(r.Kind) {
			continue
		}
		if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
			SourceTaskID: newTask.ID,
			TargetTaskID: r.TargetTaskID,
			SpaceSlug:    src.SpaceSlug,
			Kind:         r.Kind,
			CreatedAt:    nowTz,
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
		if !copyOnSpawn(r.Kind) {
			continue
		}
		// Skip incoming trigger relations: spawned tasks should not inherit
		// trigger edges, which would cause unexpected resets.
		if r.Kind == dbgen.StoredRelationKindTriggers {
			continue
		}
		if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
			SourceTaskID: r.SourceTaskID,
			TargetTaskID: newTask.ID,
			SpaceSlug:    src.SpaceSlug,
			Kind:         r.Kind,
			CreatedAt:    nowTz,
		}); err != nil {
			return 0, err
		}
	}

	// Create spawns relation: src → newTask.
	if err := q.InsertTaskRelation(ctx, dbgen.InsertTaskRelationParams{
		SourceTaskID: src.ID,
		TargetTaskID: newTask.ID,
		SpaceSlug:    src.SpaceSlug,
		Kind:         dbgen.StoredRelationKindSpawns,
		CreatedAt:    nowTz,
	}); err != nil {
		return 0, err
	}

	// Log spawn activity (system action — no user in context).
	if err := activitylog.Log(ctx, db, activitylog.Entry{
		SpaceSlug:  src.SpaceSlug,
		EntityType: activitylog.EntityTask,
		EntityID:   types.FormatTaskID(newTask.ID),
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "spawned_from", To: new(types.FormatTaskID(src.ID))},
		},
	}, now); err != nil {
		return 0, err
	}

	return newTask.ID, nil
}

// maxMissedOccurrences caps the number of tasks spawned per cron tick for a
// single fixed_accumulating task.
// allOverdueOccurrences returns all RRULE occurrences from dtstart up to and
// including `until`.
func allOverdueOccurrences(rule string, dtstart time.Time, until time.Time, loc *time.Location) ([]time.Time, error) {
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

	result := make([]time.Time, 0, len(occurrences))
	for _, occ := range occurrences {
		midnight := time.Date(occ.Year(), occ.Month(), occ.Day(), 0, 0, 0, 0, loc)
		result = append(result, midnight)
	}
	return result, nil
}

// ProcessAccumulatingTask handles one overdue fixed_accumulating task.
func ProcessAccumulatingTask(ctx context.Context, db database.DB, task dbgen.Task, now time.Time) error {
	q := dbgen.New(db)
	due := types.NewDueDate(task.DueAt, task.DueTz)
	if due == nil || !task.RecurrenceRule.Valid {
		return nil
	}

	dtstart, err := due.MidnightInTz()
	if err != nil {
		return err
	}

	loc := dtstart.Location()

	missed, err := allOverdueOccurrences(task.RecurrenceRule.String, dtstart, now, loc)
	if err != nil {
		return err
	}

	initialStatus, err := FindInitialStatus(ctx, db, task.SpaceSlug)
	if err != nil {
		return err
	}

	nowTz := types.Timestamptz(now)
	result, err := q.ConvertAccumulatingToOneOff(ctx, dbgen.ConvertAccumulatingToOneOffParams{
		UpdatedAt: nowTz,
		ID:        task.ID,
		SpaceSlug: task.SpaceSlug,
	})
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
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

	for i, missedAt := range missed {
		missedDue := types.DueDateFromLocal(missedAt, due.Tz)
		overrideAssignees := AdvanceRotation(pool, currentAssignees, i)
		if _, err := SpawnTaskFromTemplate(ctx, db, task,
			dbgen.RecurrenceTypeOneOff, pgtype.Text{}, missedDue, initialStatus, now,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	next, err := ComputeNextDueAt(dbgen.RecurrenceTypeFixedAccumulating, task.RecurrenceRule, due, now)
	if err != nil {
		return err
	}

	if next != nil {
		overrideAssignees := AdvanceRotation(pool, currentAssignees, len(missed))
		if _, err := SpawnTaskFromTemplate(ctx, db, task,
			dbgen.RecurrenceTypeFixedAccumulating, task.RecurrenceRule, next, initialStatus, now,
			overrideAssignees, pool,
		); err != nil {
			return err
		}
	}

	if err := q.DeleteRotationPool(ctx, task.ID); err != nil {
		return err
	}

	// Log the overdue advance (system action).
	return activitylog.Log(ctx, db, activitylog.Entry{
		SpaceSlug:  task.SpaceSlug,
		EntityType: activitylog.EntityTask,
		EntityID:   types.FormatTaskID(task.ID),
		Action:     activitylog.ActionUpdated,
		Details: []activitylog.Detail{
			{Field: "recurrence", From: new("fixed_accumulating"), To: new("one_off")},
		},
	}, now)
}
