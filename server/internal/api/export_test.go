package api

import "context"

// ProcessOverdueTasksForTest exposes processOverdueTasks for integration tests.
func (h *Handler) ProcessOverdueTasksForTest(ctx context.Context) {
	h.processOverdueTasks(ctx)
}
