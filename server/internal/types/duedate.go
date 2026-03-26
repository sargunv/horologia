package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// DueDate pairs a due date with its IANA timezone.
// Use *DueDate throughout internal code; nil means no due date.
type DueDate struct {
	Date time.Time
	Tz   string
}

// NewDueDate creates a *DueDate from nullable DB columns.
// Returns nil if either field is not valid.
//
// pgtype.Date stores the date as midnight UTC. We reinterpret it as midnight
// in the task's timezone so that the resulting time.Time represents the correct
// instant (e.g. 2026-06-15 midnight America/New_York = 2026-06-15T04:00:00Z).
func NewDueDate(date pgtype.Date, tz pgtype.Text) *DueDate {
	if !date.Valid || !tz.Valid {
		return nil
	}
	loc, err := time.LoadLocation(tz.String)
	if err != nil {
		return nil
	}
	d := date.Time
	midnight := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	return &DueDate{Date: midnight.UTC(), Tz: tz.String}
}

// DecomposeDueDate returns the separate DB columns for a *DueDate.
// The Date field may carry timezone information (midnight in the task's tz),
// so we extract year/month/day and store as a bare UTC date for pgtype.Date.
func DecomposeDueDate(d *DueDate) (date pgtype.Date, tz pgtype.Text) {
	if d == nil {
		return pgtype.Date{}, pgtype.Text{}
	}
	// Interpret the date in the task's timezone to get the correct calendar date.
	loc, err := time.LoadLocation(d.Tz)
	if err != nil {
		return pgtype.Date{}, pgtype.Text{}
	}
	local := d.Date.In(loc)
	bare := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	return pgtype.Date{Time: bare, Valid: true}, pgtype.Text{String: d.Tz, Valid: true}
}
