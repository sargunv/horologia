package taskengine

import (
	"context"
	"database/sql"
	"time"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// ProcessOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction. Errors for individual tasks
// are returned via the onError callback; processing continues for remaining tasks.
func ProcessOverdueTasks(ctx context.Context, db *sql.DB, onError func(taskID int64, spaceSlug string, err error)) error {
	now := time.Now()
	nowEpoch := types.EpochSecondsFrom(now)

	q := dbgen.New(db)
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, &nowEpoch)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := processOneOverdueTask(ctx, db, task, now); err != nil {
			if onError != nil {
				onError(task.ID, task.SpaceSlug, err)
			}
		}
	}
	return nil
}

// processOneOverdueTask processes a single overdue fixed_accumulating task
// in its own transaction.
func processOneOverdueTask(ctx context.Context, db *sql.DB, task dbgen.Task, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: task.ID, SpaceSlug: task.SpaceSlug})
	if err != nil {
		return err
	}

	if existing.RecurrenceType != types.RecurrenceTypeFixedAccumulating || existing.DueAt == nil || existing.DueAt.Time().After(now) {
		return nil
	}

	if err := ProcessAccumulatingTask(ctx, q, existing, now); err != nil {
		return err
	}

	return tx.Commit()
}
