package types

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatTaskID formats a task database ID as "T42".
func FormatTaskID(id int64) string {
	return "T" + strconv.FormatInt(id, 10)
}

// ParseTaskID parses a "T42" string into the numeric task ID.
func ParseTaskID(s string) (int64, error) {
	if !strings.HasPrefix(s, "T") {
		return 0, fmt.Errorf("invalid task ID %q: must start with T", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid task ID %q: %w", s, err)
	}
	return id, nil
}

// FormatRecipeID formats a recipe database ID as "R42".
func FormatRecipeID(id int64) string {
	return "R" + strconv.FormatInt(id, 10)
}

// ParseRecipeID parses an "R42" string into the numeric recipe ID.
func ParseRecipeID(s string) (int64, error) {
	if !strings.HasPrefix(s, "R") {
		return 0, fmt.Errorf("invalid recipe ID %q: must start with R", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid recipe ID %q: %w", s, err)
	}
	return id, nil
}

// FormatUserID formats a user database ID as "U42".
func FormatUserID(id int64) string {
	return "U" + strconv.FormatInt(id, 10)
}

// ParseUserID parses a "U42" string into the numeric user ID.
func ParseUserID(s string) (int64, error) {
	if !strings.HasPrefix(s, "U") {
		return 0, fmt.Errorf("invalid user ID %q: must start with U", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID %q: %w", s, err)
	}
	return id, nil
}
