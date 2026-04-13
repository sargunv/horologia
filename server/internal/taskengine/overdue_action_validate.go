package taskengine

import (
	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

// ValidateOverdueActionRule checks that an overdue action rule is compatible
// with the task's recurrence type and due date configuration.
func ValidateOverdueActionRule(
	afterDays pgtype.Int4,
	action dbgen.NullOverdueAction,
	statusName pgtype.Text,
	recurrenceType dbgen.RecurrenceType,
	hasDue bool,
) error {
	if !action.Valid {
		return nil // no rule, nothing to validate
	}
	if !hasDue {
		return types.ValidationError("overdue_action_rule requires a due date")
	}
	if action.OverdueAction == dbgen.OverdueActionAdvanceRecurrence &&
		(recurrenceType == dbgen.RecurrenceTypeOneOff || recurrenceType == dbgen.RecurrenceTypeOnDependency) {
		return types.ValidationError("advance_recurrence overdue action is only valid on recurring tasks")
	}
	if action.OverdueAction == dbgen.OverdueActionAdvanceRecurrence &&
		recurrenceType == dbgen.RecurrenceTypeFixedAccumulating {
		return types.ValidationError(
			"advance_recurrence overdue action is not supported on fixed_accumulating tasks; " +
				"the accumulating cron handles missed occurrences automatically",
		)
	}
	if action.OverdueAction == dbgen.OverdueActionSetStatus && !statusName.Valid {
		return types.ValidationError("overdue_action_rule.status is required when action is set_status")
	}
	return nil
}
