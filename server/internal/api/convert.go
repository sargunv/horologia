package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

func clampLimit(opt apigen.OptInt32) int64 {
	limit := int64(defaultLimit)
	if opt.IsSet() {
		limit = int64(opt.Value)
	}
	if limit <= 0 {
		limit = defaultLimit
	} else if limit > maxLimit {
		limit = maxLimit
	}
	return limit
}

func encodeCursor(v string) apigen.NilString {
	return apigen.NewNilString(base64.RawURLEncoding.EncodeToString([]byte(v)))
}

func decodeCursor(opt apigen.OptString) (string, error) {
	if !opt.IsSet() {
		return "", nil
	}
	b, err := base64.RawURLEncoding.DecodeString(opt.Value)
	if err != nil {
		return "", fmt.Errorf("invalid cursor: %w", err)
	}
	return string(b), nil
}

func formatTaskID(id int64) string {
	return "T" + strconv.FormatInt(id, 10)
}

func parseTaskID(s string) (int64, error) {
	if !strings.HasPrefix(s, "T") {
		return 0, fmt.Errorf("invalid task ID %q: must start with T", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid task ID %q: %w", s, err)
	}
	return id, nil
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func now() string {
	return formatTime(time.Now())
}

func spaceFromDB(s dbgen.Space) (*apigen.Space, error) {
	createdAt, err := parseTime(s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := parseTime(s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &apigen.Space{
		Slug:        s.Slug,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func taskFromDBRow(row dbgen.GetTaskWithStatusRow) (*apigen.Task, error) {
	createdAt, err := parseTime(row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := parseTime(row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}

	t := &apigen.Task{
		ID:          formatTaskID(row.ID),
		Title:       row.Title,
		Description: row.Description,
		Status: apigen.TaskStatus{
			Name:     row.StatusName,
			Category: apigen.StatusCategory(row.StatusCategory),
		},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if row.DueDate != nil {
		d, err := parseTime(*row.DueDate)
		if err != nil {
			return nil, fmt.Errorf("parse due_date: %w", err)
		}
		t.DueDate.SetTo(d)
	} else {
		t.DueDate.SetToNull()
	}

	return t, nil
}

func taskFromListRow(row dbgen.ListTasksBySpaceRow) (*apigen.Task, error) {
	return taskFromDBRow(dbgen.GetTaskWithStatusRow(row))
}

func paginate[DB any, API any](
	rows []DB,
	limit int64,
	convert func(DB) (*API, error),
	cursorOf func(DB) string,
) ([]API, apigen.NilString, error) {
	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items := make([]API, 0, len(rows))
	for _, row := range rows {
		item, err := convert(row)
		if err != nil {
			return nil, apigen.NilString{}, err
		}
		items = append(items, *item)
	}
	var next apigen.NilString
	if hasMore {
		next = encodeCursor(cursorOf(rows[len(rows)-1]))
	} else {
		next.SetToNull()
	}
	return items, next, nil
}

func dueDateToDB(opt apigen.OptNilDate) *string {
	if !opt.IsSet() || opt.IsNull() {
		return nil
	}
	s := formatTime(opt.Value)
	return &s
}

func dueDateFromExisting(existing *string, update apigen.OptNilDate) *string {
	if !update.IsSet() {
		return existing
	}
	if update.IsNull() {
		return nil
	}
	s := formatTime(update.Value)
	return &s
}
