package taskengine

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

func nullOverdueAction(a dbgen.OverdueAction) dbgen.NullOverdueAction {
	return dbgen.NullOverdueAction{OverdueAction: a, Valid: true}
}

func TestValidateOverdueActionRule(t *testing.T) {
	tests := []struct {
		name           string
		afterDays      pgtype.Int4
		action         dbgen.NullOverdueAction
		statusName     pgtype.Text
		recurrenceType dbgen.RecurrenceType
		hasDue         bool
		wantErr        bool
		errContains    string
	}{
		// ─── No rule ─────────────────────────────────────────────────────────────────
		{
			name:           "nil rule always valid",
			action:         dbgen.NullOverdueAction{Valid: false},
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         false,
			wantErr:        false,
		},
		// ─── Invalid configurations ───────────────────────────────────────────────────
		{
			name:           "advance_recurrence on one_off task",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         true,
			wantErr:        true,
			errContains:    "only valid on recurring tasks",
		},
		{
			name:           "advance_recurrence on on_dependency task",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeOnDependency,
			hasDue:         true,
			wantErr:        true,
			errContains:    "only valid on recurring tasks",
		},
		{
			name:           "rule requires due date",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeCompletionBased,
			hasDue:         false,
			wantErr:        true,
			errContains:    "requires a due date",
		},
		{
			name:           "set_status on one_off task is valid",
			action:         nullOverdueAction(dbgen.OverdueActionSetStatus),
			statusName:     pgtype.Text{String: "done", Valid: true},
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "clear_due_date on one_off task is valid",
			action:         nullOverdueAction(dbgen.OverdueActionClearDueDate),
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "set_status on on_dependency task is valid",
			action:         nullOverdueAction(dbgen.OverdueActionSetStatus),
			statusName:     pgtype.Text{String: "done", Valid: true},
			recurrenceType: dbgen.RecurrenceTypeOnDependency,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "rule on one_off requires due date",
			action:         nullOverdueAction(dbgen.OverdueActionSetStatus),
			statusName:     pgtype.Text{String: "done", Valid: true},
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         false,
			wantErr:        true,
			errContains:    "requires a due date",
		},
		{
			name:           "advance_recurrence on fixed_accumulating",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeFixedAccumulating,
			hasDue:         true,
			wantErr:        true,
			errContains:    "not supported on fixed_accumulating",
		},
		{
			name:           "set_status without status name",
			action:         nullOverdueAction(dbgen.OverdueActionSetStatus),
			statusName:     pgtype.Text{Valid: false},
			recurrenceType: dbgen.RecurrenceTypeCompletionBased,
			hasDue:         true,
			wantErr:        true,
			errContains:    "status is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOverdueActionRule(tc.afterDays, tc.action, tc.statusName, tc.recurrenceType, tc.hasDue)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
