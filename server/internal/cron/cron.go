package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sargunv/tend/server/internal/taskengine"
)

// RunAccumulatingCron ticks every interval and processes overdue
// fixed_accumulating tasks. It fires immediately on start to handle
// any backlog from server downtime. Returns when ctx is cancelled.
func RunAccumulatingCron(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger, interval time.Duration) {
	process(ctx, pool, log)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			process(ctx, pool, log)
		}
	}
}

func process(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) {
	if err := taskengine.ProcessOverdueTasks(ctx, pool, func(taskID int64, spaceSlug string, err error) {
		log.ErrorContext(ctx, "cron: process overdue task",
			"task_id", taskID,
			"space", spaceSlug,
			"error", err,
		)
	}); err != nil {
		log.ErrorContext(ctx, "cron: list overdue accumulating tasks", "error", err)
	}
}
