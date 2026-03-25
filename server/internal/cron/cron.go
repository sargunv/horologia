package cron

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// RunAccumulatingCron ticks every interval and processes overdue
// fixed_accumulating tasks. It fires immediately on start to handle
// any backlog from server downtime. Returns when ctx is cancelled.
func RunAccumulatingCron(ctx context.Context, db *sql.DB, engine *taskengine.Engine, log *slog.Logger, interval time.Duration) {
	processOverdueTasks(ctx, db, engine, log)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processOverdueTasks(ctx, db, engine, log)
		}
	}
}

// ProcessOverdueTasks finds all fixed_accumulating tasks with due_at <= now
// and processes each one in its own transaction. Exported for testing.
func ProcessOverdueTasks(ctx context.Context, db *sql.DB, engine *taskengine.Engine, log *slog.Logger) {
	processOverdueTasks(ctx, db, engine, log)
}

func processOverdueTasks(ctx context.Context, db *sql.DB, engine *taskengine.Engine, log *slog.Logger) {
	now := time.Now()
	nowEpoch := types.EpochSecondsFrom(now)

	q := dbgen.New(db)
	tasks, err := q.ListOverdueAccumulatingTasks(ctx, &nowEpoch)
	if err != nil {
		log.ErrorContext(ctx, "cron: list overdue accumulating tasks", "error", err)
		return
	}

	for _, task := range tasks {
		if err := processOneOverdueTask(ctx, db, engine, task, now); err != nil {
			log.ErrorContext(ctx, "cron: process overdue task",
				"task_id", task.ID,
				"space", task.SpaceSlug,
				"error", err,
			)
		}
	}
}

// processOneOverdueTask processes a single overdue fixed_accumulating task
// in its own transaction.
func processOneOverdueTask(ctx context.Context, db *sql.DB, engine *taskengine.Engine, task dbgen.Task, now time.Time) error {
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

	if apigen.TaskRecurrenceType(existing.RecurrenceType) != apigen.TaskRecurrenceTypeFixedAccumulating || existing.DueAt == nil || existing.DueAt.Time().After(now) {
		return nil
	}

	if err := engine.ProcessAccumulatingTask(ctx, q, existing, now); err != nil {
		return err
	}

	return tx.Commit()
}
