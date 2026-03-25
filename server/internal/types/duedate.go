package types

// DueDate pairs a due timestamp with its IANA timezone.
// Use *DueDate throughout internal code; nil means no due date.
type DueDate struct {
	At EpochSeconds
	Tz string
}

// NewDueDate creates a *DueDate from nullable DB columns.
// Returns nil if either field is nil.
func NewDueDate(at *EpochSeconds, tz *string) *DueDate {
	if at == nil || tz == nil {
		return nil
	}
	return &DueDate{At: *at, Tz: *tz}
}

// DecomposeDueDate returns the separate DB columns for a *DueDate.
func DecomposeDueDate(d *DueDate) (at *EpochSeconds, tz *string) {
	if d == nil {
		return nil, nil
	}
	return &d.At, &d.Tz
}
