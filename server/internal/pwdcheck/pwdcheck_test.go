package pwdcheck_test

import (
	"strings"
	"testing"

	"github.com/sargunv/horologia/server/internal/pwdcheck"
	"github.com/sargunv/horologia/server/internal/types"
)

func TestValidate_Length(t *testing.T) {
	tests := []struct {
		name    string
		pw      string
		wantErr string
	}{
		{"empty", "", "at least 8"},
		{"7 bytes", "1234567", "at least 8"},
		{"8 bytes", "12345678", ""},
		{"72 bytes", strings.Repeat("a", 72), ""},
		{"73 bytes", strings.Repeat("a", 73), "at most 72"},
		// Multi-byte: 3 runes × 4 bytes = 12 bytes > 8, passes min check.
		{"multibyte passes min", "🔒🔒🔒", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pwdcheck.Validate(t.Context(), tt.pw, nil)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !types.IsValidationError(err) {
				t.Fatalf("expected ValidationError, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidate_NilChecker(t *testing.T) {
	err := pwdcheck.Validate(t.Context(), "validpassword", nil)
	if err != nil {
		t.Fatalf("unexpected error with nil checker: %v", err)
	}
}
