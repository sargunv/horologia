package types

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"
)

// EpochSeconds is a time.Time stored as unix epoch seconds (INTEGER) in SQLite.
type EpochSeconds time.Time

// Now returns the current time as EpochSeconds.
func Now() EpochSeconds {
	return EpochSeconds(time.Now())
}

// EpochSecondsFrom converts a time.Time to EpochSeconds.
func EpochSecondsFrom(t time.Time) EpochSeconds {
	return EpochSeconds(t)
}

// Time returns the underlying time.Time.
func (t EpochSeconds) Time() time.Time {
	return time.Time(t)
}

// Scan implements sql.Scanner, reading an int64 from SQLite.
func (t *EpochSeconds) Scan(v any) error {
	switch v := v.(type) {
	case int64:
		*t = EpochSeconds(time.Unix(v, 0).UTC())
		return nil
	case nil:
		return errors.New("cannot scan NULL into EpochSeconds")
	default:
		return fmt.Errorf("cannot scan %T into EpochSeconds", v)
	}
}

// Value implements driver.Valuer, writing an int64 to SQLite.
func (t EpochSeconds) Value() (driver.Value, error) {
	return time.Time(t).Unix(), nil
}
