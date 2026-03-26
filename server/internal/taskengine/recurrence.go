package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/teambition/rrule-go"

	"github.com/sargunv/tend/server/internal/activitylog"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// ValidateRecurrence checks that the recurrence_type and recurrence_rule
// combination is valid. Call this on both create and update paths.
// The now parameter is used for time-dependent validation (e.g., UNTIL cap).
func ValidateRecurrence(recurrenceType dbgen.RecurrenceType, recurrenceRule pgtype.Text, now time.Time) error {
	switch recurrenceType {
	case dbgen.RecurrenceTypeOneOff, dbgen.RecurrenceTypeOnDependency:
		if recurrenceRule.Valid {
			return types.ValidationError(fmt.Sprintf("recurrence_rule must not be set for %s tasks", recurrenceType))
		}
	case dbgen.RecurrenceTypeCompletionBased, dbgen.RecurrenceTypeFixedNonAccumulating, dbgen.RecurrenceTypeFixedAccumulating:
		if !recurrenceRule.Valid {
			return types.ValidationError(fmt.Sprintf("recurrence_rule is required for %s tasks", recurrenceType))
		}
		if err := validateRRule(recurrenceRule.String, now); err != nil {
			return err
		}
	default:
		return types.ValidationError(fmt.Sprintf("invalid recurrence_type %q", recurrenceType))
	}
	return nil
}

// validateRRule parses an RRULE string and rejects unsupported properties.
func validateRRule(rule string, now time.Time) error {
	opt, err := rrule.StrToROption(rule)
	if err != nil {
		return types.ValidationError(fmt.Sprintf("invalid recurrence_rule: %v", err))
	}

	switch opt.Freq {
	case rrule.YEARLY, rrule.MONTHLY, rrule.WEEKLY, rrule.DAILY:
		// supported
	case rrule.HOURLY, rrule.MINUTELY, rrule.SECONDLY:
		return types.ValidationError("unsupported RRULE frequency: must be DAILY, WEEKLY, MONTHLY, or YEARLY")
	}

	if opt.Count > 0 {
		return types.ValidationError("RRULE COUNT is not supported; use UNTIL for finite recurrence")
	}

	if !opt.Until.IsZero() && opt.Until.After(now.AddDate(10, 0, 0)) {
		return types.ValidationError("RRULE UNTIL must not exceed 10 years in the future")
	}

	if len(opt.Byhour) > 0 {
		return types.ValidationError("unsupported RRULE property: BYHOUR")
	}
	if len(opt.Byminute) > 0 {
		return types.ValidationError("unsupported RRULE property: BYMINUTE")
	}
	if len(opt.Bysecond) > 0 {
		return types.ValidationError("unsupported RRULE property: BYSECOND")
	}

	if len(opt.Bysetpos) > 0 {
		return types.ValidationError("unsupported RRULE property: BYSETPOS")
	}
	if len(opt.Byyearday) > 0 {
		return types.ValidationError("unsupported RRULE property: BYYEARDAY")
	}
	if len(opt.Byweekno) > 0 {
		return types.ValidationError("unsupported RRULE property: BYWEEKNO")
	}

	return nil
}

// ComputeNextDueAt computes the next DueDate for a recurring task after a
// completion event. Returns nil if no next occurrence exists (rule exhausted)
// or if the recurrence type does not advance the due date. The returned
// DueDate preserves the timezone from the input (defaulting to "UTC").
//
//   - completion_based: DTSTART is now (completion time) in the task's timezone.
//   - fixed_non_accumulating / fixed_accumulating: DTSTART is the current due date
//     in the task's timezone; next occurrence after now.
//
// Precondition: ValidateRecurrence must have been called first.
func ComputeNextDueAt(recurrenceType dbgen.RecurrenceType, recurrenceRule pgtype.Text, due *types.DueDate, now time.Time) (*types.DueDate, error) {
	tz := "UTC"
	if due != nil {
		tz = due.Tz
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, types.ValidationError(fmt.Sprintf("invalid due_tz %q: %v", tz, err))
	}

	nowInTz := now.In(loc)

	var nextDate *time.Time
	switch recurrenceType {
	case dbgen.RecurrenceTypeCompletionBased:
		nextDate, err = nextRRuleOccurrence(recurrenceRule.String, nowInTz, nowInTz)
	case dbgen.RecurrenceTypeFixedNonAccumulating, dbgen.RecurrenceTypeFixedAccumulating:
		dtstart := nowInTz
		if due != nil {
			dtstart, err = due.MidnightInTz()
			if err != nil {
				return nil, err
			}
		}
		nextDate, err = nextRRuleOccurrence(recurrenceRule.String, dtstart, nowInTz)
	case dbgen.RecurrenceTypeOneOff, dbgen.RecurrenceTypeOnDependency:
		return nil, nil
	default:
		return nil, fmt.Errorf("unhandled recurrence type %q", recurrenceType)
	}
	if err != nil {
		return nil, err
	}
	if nextDate == nil {
		return nil, nil
	}
	return types.DueDateFromLocal(*nextDate, tz), nil
}

// nextRRuleOccurrence parses the rule, sets dtstart, and finds the first
// occurrence strictly after `after`. Returns nil if no future occurrence exists.
func nextRRuleOccurrence(rule string, dtstart time.Time, after time.Time) (*time.Time, error) {
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

	midnight := time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	return &midnight, nil
}

// InitialStatusFromSlice returns the name of the first initial-category status
// from a pre-fetched slice. Returns a validation error if none exists.
func InitialStatusFromSlice(statuses []dbgen.TaskStatus) (string, error) {
	for _, s := range statuses {
		if s.Category == dbgen.StatusCategoryInitial {
			return s.Name, nil
		}
	}
	return "", types.ValidationError("space has no initial status")
}

// FindInitialStatus returns the name of the first initial-category status in
// the space. Returns a validation error if no initial status exists.
func FindInitialStatus(ctx context.Context, db database.DB, spaceSlug string) (string, error) {
	q := dbgen.New(db)
	statuses, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
	if err != nil {
		return "", err
	}
	return InitialStatusFromSlice(statuses)
}

// ApplyCompletionTriggers resets tasks that have a "triggers" relation from
// the completed task. Only resets tasks with on_dependency recurrence type.
// Single-level only — ResetTaskToInitial writes directly to the DB and does not
// re-enter the update handler, so trigger cascades cannot occur.
func ApplyCompletionTriggers(ctx context.Context, db database.DB, completedTaskID int64, spaceSlug string, now pgtype.Timestamptz) error {
	q := dbgen.New(db)
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

	initialStatus, err := FindInitialStatus(ctx, db, spaceSlug)
	if err != nil {
		return err
	}

	for _, targetID := range targets {
		target, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: targetID, SpaceSlug: spaceSlug})
		if err != nil {
			return err
		}
		if target.RecurrenceType != dbgen.RecurrenceTypeOnDependency {
			continue
		}
		if err := q.ResetTaskToInitial(ctx, dbgen.ResetTaskToInitialParams{
			StatusName:   initialStatus,
			UpdatedAt:    now,
			ID:           target.ID,
			SpaceSlug:    spaceSlug,
			StatusName_2: initialStatus,
		}); err != nil {
			return err
		}

		// Log dependency trigger (system action — user who completed the trigger task is in context).
		if err := activitylog.Log(ctx, db, activitylog.Entry{
			SpaceSlug:  spaceSlug,
			EntityType: activitylog.EntityTask,
			EntityID:   types.FormatTaskID(targetID),
			Action:     activitylog.ActionUpdated,
			Details: []activitylog.Detail{
				{Field: "status", To: new(initialStatus)},
				{Field: "triggered_by", To: new(types.FormatTaskID(completedTaskID))},
			},
		}, now.Time); err != nil {
			return err
		}
	}

	return nil
}
