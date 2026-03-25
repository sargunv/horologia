package types

import (
	"database/sql/driver"
	"fmt"
)

// BoolInt is a boolean stored as INTEGER (0 or 1) in SQLite.
type BoolInt bool

// BoolIntFrom converts a bool to BoolInt.
func BoolIntFrom(b bool) BoolInt {
	return BoolInt(b)
}

// Bool returns the underlying bool.
func (b BoolInt) Bool() bool {
	return bool(b)
}

// Scan implements sql.Scanner, reading an int64 from SQLite.
func (b *BoolInt) Scan(v any) error {
	switch v := v.(type) {
	case int64:
		*b = v != 0
		return nil
	case nil:
		return fmt.Errorf("cannot scan NULL into BoolInt")
	default:
		return fmt.Errorf("cannot scan %T into BoolInt", v)
	}
}

// Value implements driver.Valuer, writing an int64 to SQLite.
func (b BoolInt) Value() (driver.Value, error) {
	if b {
		return int64(1), nil
	}
	return int64(0), nil
}
