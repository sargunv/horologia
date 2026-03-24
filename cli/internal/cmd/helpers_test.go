package cmd

import (
	"testing"
	"time"

	"github.com/sargunv/tend/server/api/gen"
)

func TestFormatAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "api error",
			err: &gen.ApiErrorStatusCode{
				StatusCode: 404,
				Response:   gen.ApiError{Code: "not_found", Message: "resource not found"},
			},
			want: "[not_found] resource not found",
		},
		{
			name: "truncates long messages",
			err: &gen.ApiErrorStatusCode{
				StatusCode: 400,
				Response:   gen.ApiError{Code: "bad_request", Message: string(make([]byte, 600))},
			},
		},
		{
			name: "strips control characters",
			err: &gen.ApiErrorStatusCode{
				StatusCode: 400,
				Response:   gen.ApiError{Code: "bad_request", Message: "hello\x00world\ttabs"},
			},
			want: "[bad_request] helloworldtabs",
		},
		{
			name: "preserves newlines",
			err: &gen.ApiErrorStatusCode{
				StatusCode: 400,
				Response:   gen.ApiError{Code: "bad_request", Message: "line1\nline2"},
			},
			want: "[bad_request] line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAPIError(tt.err)
			if tt.want != "" && got.Error() != tt.want {
				t.Errorf("FormatAPIError() = %q, want %q", got.Error(), tt.want)
			}
			// For the truncation test, just verify length.
			if tt.name == "truncates long messages" {
				if len(got.Error()) > 530 {
					t.Errorf("FormatAPIError() message too long: %d chars", len(got.Error()))
				}
			}
		})
	}
}

func TestParseDueDate(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		wantDay int
	}{
		{"2026-04-01", false, 1},
		{"2026-12-25", false, 25},
		{"", true, 0},
		{"04-01-2026", true, 0},
		{"not-a-date", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDueDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDueDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Day() != tt.wantDay {
				t.Errorf("parseDueDate(%q).Day() = %d, want %d", tt.input, got.Day(), tt.wantDay)
			}
		})
	}
}

func TestFormatAssignees(t *testing.T) {
	tests := []struct {
		name     string
		ids      []string
		fallback string
		maxShow  int
		want     string
	}{
		{"empty", nil, "-", 0, "-"},
		{"one", []string{"U1"}, "-", 0, "U1"},
		{"multiple", []string{"U1", "U2", "U3"}, "-", 0, "U1, U2, U3"},
		{"truncated", []string{"U1", "U2", "U3", "U4"}, "-", 2, "U1, U2 +2 more"},
		{"exact limit", []string{"U1", "U2"}, "-", 2, "U1, U2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := gen.Task{AssigneeIds: tt.ids}
			got := formatAssignees(task, tt.fallback, tt.maxShow)
			if got != tt.want {
				t.Errorf("formatAssignees() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDue(t *testing.T) {
	t.Run("with date", func(t *testing.T) {
		d := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		task := gen.Task{}
		task.Due.SetTo(gen.TaskDue{At: d, Timezone: "UTC"})
		got := formatDue(task, "-")
		if got != "2026-04-01" {
			t.Errorf("formatDue() = %q, want %q", got, "2026-04-01")
		}
	})

	t.Run("without date", func(t *testing.T) {
		task := gen.Task{}
		task.Due.SetToNull()
		got := formatDue(task, "n/a")
		if got != "n/a" {
			t.Errorf("formatDue() = %q, want %q", got, "n/a")
		}
	})
}
