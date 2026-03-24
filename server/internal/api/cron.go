package api

import (
	"context"
	"time"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// RunAccumulatingCron ticks every interval and processes overdue
// fixed_accumulating tasks. It fires immediately on start to handle
// any backlog from server downtime. Returns when ctx is cancelled.
func (h *Handler) RunAccumulatingCron(ctx context.Context, interval time.Duration) {
	// Process immediately on startup.
	h.processOverdueTasks(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.processOverdueTasks(ctx)
		}
	}
}

// processOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction.
func (h *Handler) processOverdueTasks(ctx context.Context) {
	now := time.Now()
	nowEpoch := types.EpochSecondsFrom(now)

	q := dbgen.New(h.DB)
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, &nowEpoch)
	if err != nil {
		h.Log.ErrorContext(ctx, "cron: list overdue accumulating tasks", "error", err)
		return
	}

	for _, task := range tasks {
		if err := h.processOneOverdueTask(ctx, task, now); err != nil {
			h.Log.ErrorContext(ctx, "cron: process overdue task",
				"task_id", task.ID,
				"space", task.SpaceSlug,
				"error", err,
			)
		}
	}
}

// processOneOverdueTask processes a single overdue fixed_accumulating task
// in its own transaction. Re-fetches the task inside the transaction to
// avoid TOCTOU races with concurrent HTTP requests.
func (h *Handler) processOneOverdueTask(ctx context.Context, task dbgen.Task, now time.Time) error {
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	// Re-fetch inside transaction to verify the task is still overdue and
	// still fixed_accumulating (could have been completed or changed).
	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: task.ID, SpaceSlug: task.SpaceSlug})
	if err != nil {
		return err
	}

	if existing.RecurrenceType != "fixed_accumulating" || existing.DueAt == nil || existing.DueAt.Time().After(now) {
		return nil // stale, skip
	}

	if err := h.processAccumulatingTask(ctx, q, existing, now); err != nil {
		return err
	}

	return tx.Commit()
}
