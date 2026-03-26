package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// DueDate pairs a calendar date with its IANA timezone.
// Use *DueDate throughout internal code; nil means no due date.
type DueDate struct {
	Date pgtype.Date
	Tz   string
}

// NewDueDate creates a *DueDate from nullable DB columns.
// Returns nil if either field is not valid.
func NewDueDate(date pgtype.Date, tz pgtype.Text) *DueDate {
	if !date.Valid || !tz.Valid {
		return nil
	}
	return &DueDate{Date: date, Tz: tz.String}
}

// DueDateFromLocal creates a *DueDate from a time in a local timezone,
// extracting just the calendar date. Used when RRULE computation returns
// a midnight-in-timezone value that needs to be stored as a plain date.
func DueDateFromLocal(local time.Time, tz string) *DueDate {
	return &DueDate{
		Date: pgtype.Date{
			Time:  time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		Tz: tz,
	}
}

// MidnightInTz returns the date as midnight in the task's timezone.
// Used when RRULE or overdue logic needs a timezone-aware instant.
func (d *DueDate) MidnightInTz() (time.Time, error) {
	loc, err := time.LoadLocation(d.Tz)
	if err != nil {
		return time.Time{}, err
	}
	t := d.Date.Time
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
}

// DecomposeDueDate returns the separate DB columns for a *DueDate.
func DecomposeDueDate(d *DueDate) (date pgtype.Date, tz pgtype.Text) {
	if d == nil {
		return pgtype.Date{}, pgtype.Text{}
	}
	return d.Date, pgtype.Text{String: d.Tz, Valid: true}
}
