package pwdcheck

import (
	"context"

	"github.com/sargunv/horologia/server/internal/types"
)

const (
	MinLength = 8
	MaxLength = 72 // bcrypt silently truncates beyond 72 bytes
)

// Checker checks whether a password appears in known breached datasets.
// A nil Checker is treated as disabled (always passes).
type Checker interface {
	Check(ctx context.Context, password string) error
}

// Validate checks the password against length requirements and the optional
// breach checker. Returns a types.ValidationError on failure.
func Validate(ctx context.Context, password string, checker Checker) error {
	// Use byte length because bcrypt operates on bytes and its 72-byte limit
	// is a byte limit. The minimum is also in bytes for consistency.
	n := len([]byte(password))
	if n < MinLength {
		return types.ValidationError("password must be at least 8 characters")
	}
	if n > MaxLength {
		return types.ValidationError("password must be at most 72 characters")
	}
	if checker != nil {
		if err := checker.Check(ctx, password); err != nil {
			return err
		}
	}
	return nil
}
