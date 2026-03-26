package taskengine

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// ProcessOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction. Errors for individual tasks
// are returned via the onError callback; processing continues for remaining tasks.
func ProcessOverdueTasks(ctx context.Context, pool *pgxpool.Pool, onError func(taskID int64, spaceSlug string, err error)) error {
	now := time.Now()

	q := dbgen.New(pool)
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, pgtype.Date{Time: now, Valid: true})
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if err := processOneOverdueTask(ctx, pool, task, now); err != nil {
			if onError != nil {
				onError(task.ID, task.SpaceSlug, err)
			}
		}
	}
	return nil
}

// processOneOverdueTask processes a single overdue fixed_accumulating task
// in its own transaction.
func processOneOverdueTask(ctx context.Context, pool *pgxpool.Pool, task dbgen.Task, now time.Time) error {
	tx, err := pool.Begin(ctx)
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

	if err := ProcessAccumulatingTask(ctx, q, existing, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
