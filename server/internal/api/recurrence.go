package api

import (
	"context"
	"fmt"
	"time"

	"github.com/teambition/rrule-go"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// validateRecurrence checks that the recurrence_type and recurrence_rule
// combination is valid. Call this on both create and update paths.
// The now parameter is used for time-dependent validation (e.g., UNTIL cap).
func validateRecurrence(recurrenceType string, recurrenceRule *string, now time.Time) error {
	switch recurrenceType {
	case "one_off", "on_dependency":
		if recurrenceRule != nil {
			return badRequest(fmt.Sprintf("recurrence_rule must not be set for %s tasks", recurrenceType))
		}
	case "completion_based", "fixed_non_accumulating", "fixed_accumulating":
		if recurrenceRule == nil {
			return badRequest(fmt.Sprintf("recurrence_rule is required for %s tasks", recurrenceType))
		}
		if err := validateRRule(*recurrenceRule, now); err != nil {
			return err
		}
	default:
		return badRequest(fmt.Sprintf("invalid recurrence_type %q", recurrenceType))
	}
	return nil
}

// validateRRule parses an RRULE string and rejects unsupported properties.
// The now parameter is used for time-dependent caps (e.g., UNTIL).
func validateRRule(rule string, now time.Time) error {
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return badRequest(fmt.Sprintf("invalid recurrence_rule: %v", err))
	}

	// Reject sub-day frequencies.
	switch opt.Freq {
	case rrule.YEARLY, rrule.MONTHLY, rrule.WEEKLY, rrule.DAILY:
		// supported
	case rrule.HOURLY, rrule.MINUTELY, rrule.SECONDLY:
		return badRequest("unsupported RRULE frequency: must be DAILY, WEEKLY, MONTHLY, or YEARLY")
	}

	// Reject COUNT — no longer needed for performance (DTSTART is updated on
	// each completion), and its semantics are confusing when DTSTART resets.
	// Use UNTIL for finite recurrence instead.
	if opt.Count > 0 {
		return badRequest("RRULE COUNT is not supported; use UNTIL for finite recurrence")
	}

	// Cap UNTIL to prevent excessive iteration.
	if !opt.Until.IsZero() && opt.Until.After(now.AddDate(10, 0, 0)) {
		return badRequest("RRULE UNTIL must not exceed 10 years in the future")
	}

	// Reject sub-day properties (meaningless for date-only due dates).
	if len(opt.Byhour) > 0 {
		return badRequest("unsupported RRULE property: BYHOUR")
	}
	if len(opt.Byminute) > 0 {
		return badRequest("unsupported RRULE property: BYMINUTE")
	}
	if len(opt.Bysecond) > 0 {
		return badRequest("unsupported RRULE property: BYSECOND")
	}

	// Reject complex positional properties.
	if len(opt.Bysetpos) > 0 {
		return badRequest("unsupported RRULE property: BYSETPOS")
	}
	if len(opt.Byyearday) > 0 {
		return badRequest("unsupported RRULE property: BYYEARDAY")
	}
	if len(opt.Byweekno) > 0 {
		return badRequest("unsupported RRULE property: BYWEEKNO")
	}

	return nil
}

// computeNextDueAt computes the next due epoch seconds for a recurring task
// after a completion event. Returns nil if no next occurrence exists (rule
// exhausted) or if the recurrence type does not advance the due date.
//
//   - completion_based: DTSTART is now (completion time) in the task's timezone.
//   - fixed_non_accumulating / fixed_accumulating: DTSTART is the current due_at
//     in the task's timezone; next occurrence after now.
//
// Precondition: validateRecurrence must have been called first.
func computeNextDueAt(recurrenceType string, recurrenceRule *string, dueAt *types.EpochSeconds, dueTz *string, now time.Time) (*types.EpochSeconds, error) {
	loc := time.UTC
	if dueTz != nil {
		var err error
		loc, err = time.LoadLocation(*dueTz)
		if err != nil {
			return nil, badRequest(fmt.Sprintf("invalid due_tz %q: %v", *dueTz, err))
		}
	}

	nowInTz := now.In(loc)

	switch recurrenceType {
	case "completion_based":
		// DTSTART = now in the task's timezone. Next occurrence strictly after now.
		return nextRRuleOccurrence(*recurrenceRule, nowInTz, nowInTz)
	case "fixed_non_accumulating", "fixed_accumulating":
		// DTSTART = current due date in the task's timezone.
		dtstart := nowInTz
		if dueAt != nil {
			dtstart = dueAt.Time().In(loc)
		}
		return nextRRuleOccurrence(*recurrenceRule, dtstart, nowInTz)
	default:
		return nil, nil
	}
}

// nextRRuleOccurrence parses the rule, sets dtstart, and finds the first
// occurrence strictly after `after`. Returns nil if no future occurrence exists.
func nextRRuleOccurrence(rule string, dtstart time.Time, after time.Time) (*types.EpochSeconds, error) {
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return nil, fmt.Errorf("invalid recurrence_rule: %w", err)
	}
	opt.Dtstart = dtstart

	rr, err := rrule.NewRRule(*opt)
	if err != nil {
		return nil, fmt.Errorf("invalid recurrence_rule: %w", err)
	}

	next := rr.After(after, false)
	if next.IsZero() {
		return nil, nil
	}

	// Truncate to midnight in the occurrence's timezone to keep date-level precision.
	midnight := time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	es := types.EpochSecondsFrom(midnight)
	return &es, nil
}

// initialStatusFromSlice returns the name of the first initial-category status
// from a pre-fetched slice. Returns a badRequest error if none exists.
func initialStatusFromSlice(statuses []dbgen.TaskStatus) (string, error) {
	for _, s := range statuses {
		if s.Category == "initial" {
			return s.Name, nil
		}
	}
	return "", badRequest("space has no initial status")
}

// findInitialStatus returns the name of the first initial-category status in
// the space. Returns a badRequest error if no initial status exists.
func findInitialStatus(ctx context.Context, q *dbgen.Queries, spaceSlug string) (string, error) {
	statuses, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
	if err != nil {
		return "", err
	}
	return initialStatusFromSlice(statuses)
}

// applyCompletionTriggers resets tasks that have a "triggers" relation from
// the completed task. Only resets tasks with on_dependency recurrence type.
// Single-level only — ResetTaskToInitial writes directly to the DB and does not
// re-enter the update handler, so trigger cascades cannot occur.
// The now parameter is forwarded to ResetTaskToInitial as the updated_at timestamp.
func (h *Handler) applyCompletionTriggers(ctx context.Context, q *dbgen.Queries, completedTaskID int64, spaceSlug string, now types.EpochSeconds) error {
	targets, err := q.ListTriggerTargets(ctx, dbgen.ListTriggerTargetsParams{
		SourceTaskID: completedTaskID,
		SpaceSlug:    spaceSlug,
	})
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		return nil
	}

	initialStatus, err := findInitialStatus(ctx, q, spaceSlug)
	if err != nil {
		return err
	}

	for _, targetID := range targets {
		target, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: targetID, SpaceSlug: spaceSlug})
		if err != nil {
			return err
		}
		if target.RecurrenceType != "on_dependency" {
			continue
		}
		// The SQL query includes a status_name != ? guard to make the
		// check-and-write atomic, avoiding TOCTOU issues. If the task is
		// already at the initial status, the UPDATE is a no-op.
		if err := q.ResetTaskToInitial(ctx, dbgen.ResetTaskToInitialParams{
			StatusName:   initialStatus,
			UpdatedAt:    now,
			ID:           target.ID,
			SpaceSlug:    spaceSlug,
			StatusName_2: initialStatus,
		}); err != nil {
			return err
		}
	}

	return nil
}
