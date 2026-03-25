package taskengine

import (
	"context"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// AdvanceRotation derives the next assignee from the rotation pool.
//
// It finds the first element of currentAssignees that appears in pool,
// then advances by (1 + step) positions (wrapping). If no current assignee
// is in the pool, it returns pool[step % len(pool)].
//
// step is 0 for normal completion or a single spawn, and N for the Nth
// successive spawn in the same operation (e.g. missed cron occurrences).
// Returns nil if the pool is empty.
func AdvanceRotation(pool []int64, currentAssignees []int64, step int) []int64 {
	if len(pool) == 0 {
		return nil
	}
	poolIdx := make(map[int64]int, len(pool))
	for i, uid := range pool {
		poolIdx[uid] = i
	}
	baseIdx := -1
	for _, uid := range currentAssignees {
		if idx, ok := poolIdx[uid]; ok {
			baseIdx = idx
			break
		}
	}
	var next int
	if baseIdx == -1 {
		next = step % len(pool)
	} else {
		next = (baseIdx + 1 + step) % len(pool)
	}
	return []int64{pool[next]}
}

// ApplyPoolRotation reads the rotation pool and current assignees for a task,
// computes the next assignee, and writes it. No-op if the pool is empty.
// Must be called within a transaction.
func ApplyPoolRotation(ctx context.Context, q *dbgen.Queries, taskID int64, now types.EpochSeconds) error {
	pool, err := q.ListRotationPoolByTask(ctx, taskID)
	if err != nil {
		return err
	}
	if len(pool) == 0 {
		return nil
	}
	currentAssignees, err := q.ListAssigneeUserIDsByTask(ctx, taskID)
	if err != nil {
		return err
	}
	nextAssignees := AdvanceRotation(pool, currentAssignees, 0)
	if err := q.DeleteTaskAssignees(ctx, taskID); err != nil {
		return err
	}
	for _, uid := range nextAssignees {
		if err := q.InsertTaskAssignee(ctx, dbgen.InsertTaskAssigneeParams{
			TaskID:    taskID,
			UserID:    uid,
			CreatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}
