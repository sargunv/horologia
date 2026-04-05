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

func nullText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
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
		// ─── Valid configurations ─────────────────────────────────────────────────────
		{
			name:           "advance_recurrence on completion_based with due",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeCompletionBased,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "advance_recurrence on fixed_non_accumulating with due",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeFixedNonAccumulating,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "set_status on completion_based with status and due",
			action:         nullOverdueAction(dbgen.OverdueActionSetStatus),
			statusName:     nullText("overdue"),
			recurrenceType: dbgen.RecurrenceTypeCompletionBased,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "clear_due_date on fixed_non_accumulating with due",
			action:         nullOverdueAction(dbgen.OverdueActionClearDueDate),
			recurrenceType: dbgen.RecurrenceTypeFixedNonAccumulating,
			hasDue:         true,
			wantErr:        false,
		},
		{
			name:           "clear_due_date on fixed_accumulating with due",
			action:         nullOverdueAction(dbgen.OverdueActionClearDueDate),
			recurrenceType: dbgen.RecurrenceTypeFixedAccumulating,
			hasDue:         true,
			wantErr:        false,
		},
		// ─── Invalid configurations ───────────────────────────────────────────────────
		{
			name:           "rule on one_off task",
			action:         nullOverdueAction(dbgen.OverdueActionAdvanceRecurrence),
			recurrenceType: dbgen.RecurrenceTypeOneOff,
			hasDue:         true,
			wantErr:        true,
			errContains:    "only valid on recurring tasks",
		},
		{
			name:           "rule on on_dependency task",
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
