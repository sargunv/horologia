package taskengine

import (
	"context"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// CompletionResult holds the possibly-mutated fields after evaluating a
// status transition for completion logic.
type CompletionResult struct {
	Status          string
	RecurrenceType  string
	RecurrenceRule  *string
	DueAt           *types.EpochSeconds
	DueTz           *string
	LastCompletedAt *types.EpochSeconds
	JustCompleted   bool
}

// HandleCompletionTransition evaluates whether a status change constitutes a
// completion transition and, if so, applies recurrence advancement, task
// spawning, and pool rotation. Returns the (possibly mutated) field values
// that the caller should write to the task row.
//
// If the status did not change or the transition is not a completion, the
// input values are returned unchanged with JustCompleted=false.
func (e *Engine) HandleCompletionTransition(
	ctx context.Context,
	q *dbgen.Queries,
	existing dbgen.Task,
	newStatus string,
	recurrenceType string,
	recurrenceRule *string,
	dueAt *types.EpochSeconds,
	dueTz *string,
	lastCompletedAt *types.EpochSeconds,
	spaceSlug string,
	taskID int64,
	now types.EpochSeconds,
) (*CompletionResult, error) {
	result := &CompletionResult{
		Status:          newStatus,
		RecurrenceType:  recurrenceType,
		RecurrenceRule:  recurrenceRule,
		DueAt:           dueAt,
		DueTz:           dueTz,
		LastCompletedAt: lastCompletedAt,
	}

	if newStatus == existing.StatusName {
		return result, nil
	}

	statuses, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
	if err != nil {
		return nil, err
	}

	categoryOf := func(name string) string {
		for _, s := range statuses {
			if s.Name == name {
				return s.Category
			}
		}
		return ""
	}

	oldIsCompletion := categoryOf(existing.StatusName) == "completion"
	newIsCompletion := categoryOf(newStatus) == "completion"

	if oldIsCompletion || !newIsCompletion {
		return result, nil
	}

	result.JustCompleted = true
	result.LastCompletedAt = &now

	switch recurrenceType {
	case "completion_based", "fixed_non_accumulating":
		next, err := ComputeNextDueAt(recurrenceType, recurrenceRule, existing.DueAt, existing.DueTz, now.Time())
		if err != nil {
			return nil, err
		}
		if next != nil {
			result.DueAt = next
			if result.DueTz == nil {
				utc := "UTC"
				result.DueTz = &utc
			}
			initialStatus, err := InitialStatusFromSlice(statuses)
			if err != nil {
				return nil, err
			}
			result.Status = initialStatus
		}

	case "fixed_accumulating":
		next, err := ComputeNextDueAt(recurrenceType, recurrenceRule, existing.DueAt, existing.DueTz, now.Time())
		if err != nil {
			return nil, err
		}
		if next != nil {
			initialStatus, err := InitialStatusFromSlice(statuses)
			if err != nil {
				return nil, err
			}
			pool, err := q.ListRotationPoolByTask(ctx, taskID)
			if err != nil {
				return nil, err
			}
			currentAssignees, err := q.ListAssigneeUserIDsByTask(ctx, taskID)
			if err != nil {
				return nil, err
			}
			overrideAssignees := AdvanceRotation(pool, currentAssignees, 0)
			if _, err := e.SpawnTaskFromTemplate(ctx, q, existing,
				"fixed_accumulating", recurrenceRule, next, initialStatus, now,
				overrideAssignees, pool,
			); err != nil {
				return nil, err
			}
		}
		result.RecurrenceType = "one_off"
		result.RecurrenceRule = nil
		if err := q.DeleteRotationPool(ctx, taskID); err != nil {
			return nil, err
		}
	}

	// Apply pool rotation for recurrence types that reset in place.
	// fixed_accumulating is handled above (rotation passed to spawned task).
	if result.RecurrenceType == "completion_based" || result.RecurrenceType == "fixed_non_accumulating" {
		if err := ApplyPoolRotation(ctx, q, taskID, now); err != nil {
			return nil, err
		}
	}

	return result, nil
}
