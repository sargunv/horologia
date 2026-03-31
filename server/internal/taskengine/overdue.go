package taskengine

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// ProcessOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction. Errors for individual tasks
// are returned via the onError callback; processing continues for remaining tasks.
func ProcessOverdueTasks(ctx context.Context, db database.DB, now time.Time, onError func(taskID int64, spaceSlug string, err error)) error {
	q := dbgen.New(db)
	// The query caps results at 100 rows per tick for backpressure;
	// remaining overdue tasks will be picked up in subsequent cron ticks.
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, pgtype.Date{Time: now, Valid: true})
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
func processOneOverdueTask(ctx context.Context, db database.DB, task dbgen.Task, now time.Time) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: task.ID, SpaceSlug: task.SpaceSlug})
	if err != nil {
		return err
	}

	if existing.RecurrenceType != dbgen.RecurrenceTypeFixedAccumulating || !existing.DueAt.Valid || existing.DueAt.Time.After(now) {
		return nil
	}

	if err := ProcessAccumulatingTask(ctx, tx, existing, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
