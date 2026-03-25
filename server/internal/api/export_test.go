package api

import (
	"context"

	"github.com/sargunv/tend/server/internal/cron"
)

// ProcessOverdueTasksForTest exposes ProcessOverdueTasks for integration tests.
func (h *Handler) ProcessOverdueTasksForTest(ctx context.Context) {
	cron.ProcessOverdueTasks(ctx, h.DB, h.Engine, h.Log)
}
