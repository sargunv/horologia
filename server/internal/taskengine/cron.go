package taskengine

import (
	"context"
	"time"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// RunAccumulatingCron ticks every interval and processes overdue
// fixed_accumulating tasks. It fires immediately on start to handle
// any backlog from server downtime. Returns when ctx is cancelled.
func (e *Engine) RunAccumulatingCron(ctx context.Context, interval time.Duration) {
	e.ProcessOverdueTasks(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.ProcessOverdueTasks(ctx)
		}
	}
}

// ProcessOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction.
func (e *Engine) ProcessOverdueTasks(ctx context.Context) {
	now := time.Now()
	nowEpoch := types.EpochSecondsFrom(now)

	q := dbgen.New(e.DB)
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, &nowEpoch)
	if err != nil {
		e.Log.ErrorContext(ctx, "cron: list overdue accumulating tasks", "error", err)
		return
	}

	for _, task := range tasks {
		if err := e.processOneOverdueTask(ctx, task, now); err != nil {
			e.Log.ErrorContext(ctx, "cron: process overdue task",
				"task_id", task.ID,
				"space", task.SpaceSlug,
				"error", err,
			)
		}
	}
}

// processOneOverdueTask processes a single overdue fixed_accumulating task
// in its own transaction.
func (e *Engine) processOneOverdueTask(ctx context.Context, task dbgen.Task, now time.Time) error {
	tx, err := e.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: task.ID, SpaceSlug: task.SpaceSlug})
	if err != nil {
		return err
	}

	if existing.RecurrenceType != "fixed_accumulating" || existing.DueAt == nil || existing.DueAt.Time().After(now) {
		return nil
	}

	if err := e.processAccumulatingTask(ctx, q, existing, now); err != nil {
		return err
	}

	return tx.Commit()
}
