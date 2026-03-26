package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// CompletionResult holds the possibly-mutated fields after evaluating a
// status transition for completion logic.
type CompletionResult struct {
	Status          string
	RecurrenceType  dbgen.RecurrenceType
	RecurrenceRule  pgtype.Text
	Due             *types.DueDate
	LastCompletedAt pgtype.Timestamptz
	JustCompleted   bool
}

// HandleCompletionTransition evaluates whether a status change constitutes a
// completion transition and, if so, applies recurrence advancement, task
// spawning, and pool rotation. Returns the (possibly mutated) field values
// that the caller should write to the task row.
//
// If the status did not change or the transition is not a completion, the
// input values are returned unchanged with JustCompleted=false.
func HandleCompletionTransition(
	ctx context.Context,
	q *dbgen.Queries,
	existing dbgen.Task,
	newStatus string,
	recurrenceType dbgen.RecurrenceType,
	recurrenceRule pgtype.Text,
	due *types.DueDate,
	lastCompletedAt pgtype.Timestamptz,
	spaceSlug string,
	taskID int64,
	now time.Time,
) (*CompletionResult, error) {
	result := &CompletionResult{
		Status:          newStatus,
		RecurrenceType:  recurrenceType,
		RecurrenceRule:  recurrenceRule,
		Due:             due,
		LastCompletedAt: lastCompletedAt,
	}

	if newStatus == existing.StatusName {
		return result, nil
	}

	statuses, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
	if err != nil {
		return nil, err
	}

	categoryOf := func(name string) dbgen.StatusCategory {
		for _, s := range statuses {
			if s.Name == name {
				return s.Category
			}
		}
		return ""
	}

	oldIsCompletion := categoryOf(existing.StatusName) == dbgen.StatusCategoryCompletion
	newIsCompletion := categoryOf(newStatus) == dbgen.StatusCategoryCompletion

	if oldIsCompletion || !newIsCompletion {
		return result, nil
	}

	result.JustCompleted = true
	result.LastCompletedAt = types.Timestamptz(now)

	existingDue := types.NewDueDate(existing.DueAt, existing.DueTz)

	switch recurrenceType {
	case dbgen.RecurrenceTypeOneOff, dbgen.RecurrenceTypeOnDependency:
		// no special completion behavior

	case dbgen.RecurrenceTypeCompletionBased, dbgen.RecurrenceTypeFixedNonAccumulating:
		next, err := ComputeNextDueAt(recurrenceType, recurrenceRule, existingDue, now)
		if err != nil {
			return nil, err
		}
		if next != nil {
			result.Due = next
			initialStatus, err := InitialStatusFromSlice(statuses)
			if err != nil {
				return nil, err
			}
			result.Status = initialStatus
		}

	case dbgen.RecurrenceTypeFixedAccumulating:
		next, err := ComputeNextDueAt(recurrenceType, recurrenceRule, existingDue, now)
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
			if _, err := SpawnTaskFromTemplate(ctx, q, existing,
				dbgen.RecurrenceTypeFixedAccumulating, recurrenceRule, next, initialStatus, now,
				overrideAssignees, pool,
			); err != nil {
				return nil, err
			}
		}
		result.RecurrenceType = dbgen.RecurrenceTypeOneOff
		result.RecurrenceRule = pgtype.Text{}
		if err := q.DeleteRotationPool(ctx, taskID); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unhandled recurrence type %q in completion transition", recurrenceType)
	}

	// Apply pool rotation for recurrence types that reset in place.
	// fixed_accumulating is handled above (rotation passed to spawned task).
	if result.RecurrenceType == dbgen.RecurrenceTypeCompletionBased || result.RecurrenceType == dbgen.RecurrenceTypeFixedNonAccumulating {
		if err := ApplyPoolRotation(ctx, q, taskID, now); err != nil {
			return nil, err
		}
	}

	return result, nil
}
