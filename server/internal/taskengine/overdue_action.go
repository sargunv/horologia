package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sargunv/tend/server/internal/activitylog"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// ProcessOverdueActionTasks finds all recurring tasks whose overdue action
// grace period has elapsed and applies the configured action to each one.
// Errors are reported via onError; processing continues for remaining tasks.
func ProcessOverdueActionTasks(
	ctx context.Context,
	db database.DB,
	now time.Time,
	onError func(taskID int64, spaceSlug string, err error),
) error {
	q := dbgen.New(db)
	today := pgtype.Date{Time: now.UTC().Truncate(24 * time.Hour), Valid: true}
	tasks, err := q.ListTasksWithOverdueActionDue(ctx, today)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := applyOverdueAction(ctx, db, task, now); err != nil {
			if onError != nil {
				onError(task.ID, task.SpaceSlug, err)
			}
		}
	}
	return nil
}

func applyOverdueAction(ctx context.Context, db database.DB, task dbgen.Task, now time.Time) error {
	if !task.OverdueAction.Valid || !task.DueAt.Valid {
		return nil
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	nowTz := types.Timestamptz(now)

	switch task.OverdueAction.OverdueAction {
	case dbgen.OverdueActionAdvanceRecurrence:
		err = applyAdvanceRecurrence(ctx, tx, task, now, nowTz)
	case dbgen.OverdueActionSetStatus:
		err = applySetStatus(ctx, tx, task, now, nowTz)
	case dbgen.OverdueActionClearDueDate:
		err = applyClearDueDate(ctx, tx, task, now, nowTz)
	default:
		return fmt.Errorf("unknown overdue action %q on task %d", task.OverdueAction.OverdueAction, task.ID)
	}
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func applyAdvanceRecurrence(
	ctx context.Context, db dbgen.DBTX, task dbgen.Task, now time.Time, nowTz pgtype.Timestamptz,
) error {
	q := dbgen.New(db)
	due := types.NewDueDate(task.DueAt, task.DueTz)
	next, err := ComputeNextDueAt(task.RecurrenceType, task.RecurrenceRule, due, now)
	if err != nil {
		return err
	}
	newDueAt, newDueTz := types.DecomposeDueDate(next)
	result, err := q.UpdateTaskOverdueActionAdvanceRecurrence(ctx, dbgen.UpdateTaskOverdueActionAdvanceRecurrenceParams{
		DueAt:     newDueAt,
		DueTz:     newDueTz,
		UpdatedAt: nowTz,
		ID:        task.ID,
		SpaceSlug: task.SpaceSlug,
	})
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return nil // concurrent update won
	}
	fromVal := task.DueAt.Time.Format("2006-01-02")
	var toVal *string
	if next != nil {
		s := next.Date.Time.Format("2006-01-02")
		toVal = &s
	}
	return activitylog.Log(ctx, db, activitylog.Entry{
		SpaceSlug:  task.SpaceSlug,
		EntityType: activitylog.EntityTask,
		EntityID:   types.FormatTaskID(task.ID),
		Action:     activitylog.ActionUpdated,
		Details: []activitylog.Detail{
			{Field: "due", From: &fromVal, To: toVal},
		},
	}, now)
}

func applySetStatus(
	ctx context.Context, db dbgen.DBTX, task dbgen.Task, now time.Time, nowTz pgtype.Timestamptz,
) error {
	if !task.OverdueActionStatus.Valid {
		return fmt.Errorf("set_status action on task %d has no status configured", task.ID)
	}
	q := dbgen.New(db)
	// Validate the target status still exists in the space.
	statuses, err := q.ListTaskStatusesBySpace(ctx, task.SpaceSlug)
	if err != nil {
		return err
	}
	found := false
	for _, s := range statuses {
		if s.Name == task.OverdueActionStatus.String {
			found = true
			break
		}
	}
	if !found {
		// Status was deleted; skip silently rather than crashing the cron.
		return nil
	}
	result, err := q.UpdateTaskOverdueActionSetStatus(ctx, dbgen.UpdateTaskOverdueActionSetStatusParams{
		StatusName: task.OverdueActionStatus.String,
		UpdatedAt:  nowTz,
		ID:         task.ID,
		SpaceSlug:  task.SpaceSlug,
	})
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return nil
	}
	fromVal := task.StatusName
	toVal := task.OverdueActionStatus.String
	return activitylog.Log(ctx, db, activitylog.Entry{
		SpaceSlug:  task.SpaceSlug,
		EntityType: activitylog.EntityTask,
		EntityID:   types.FormatTaskID(task.ID),
		Action:     activitylog.ActionUpdated,
		Details: []activitylog.Detail{
			{Field: "status", From: &fromVal, To: &toVal},
		},
	}, now)
}

func applyClearDueDate(
	ctx context.Context, db dbgen.DBTX, task dbgen.Task, now time.Time, nowTz pgtype.Timestamptz,
) error {
	q := dbgen.New(db)
	result, err := q.UpdateTaskOverdueActionClearDueDate(ctx, dbgen.UpdateTaskOverdueActionClearDueDateParams{
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
	fromVal := task.DueAt.Time.Format("2006-01-02")
	return activitylog.Log(ctx, db, activitylog.Entry{
		SpaceSlug:  task.SpaceSlug,
		EntityType: activitylog.EntityTask,
		EntityID:   types.FormatTaskID(task.ID),
		Action:     activitylog.ActionUpdated,
		Details: []activitylog.Detail{
			{Field: "due", From: &fromVal},
		},
	}, now)
}
