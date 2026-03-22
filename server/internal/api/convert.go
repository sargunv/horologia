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

func decodeCursorInt64(opt apigen.OptString) (int64, error) {
	s, err := decodeCursor(opt)
	if err != nil {
		return 0, err
	}
	if s == "" {
		return 0, nil
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid cursor")
	}
	return id, nil
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

// parseAssigneeIDs parses a GROUP_CONCAT result (comma-separated int64s) into
// formatted user ID strings. An empty string returns an empty slice.
func parseAssigneeIDs(raw any) ([]string, error) {
	var s string
	switch v := raw.(type) {
	case nil:
		return []string{}, nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return nil, fmt.Errorf("unexpected assignee_ids type %T", raw)
	}
	if s == "" {
		return []string{}, nil
	}
	parts := strings.Split(s, ",")
	ids := make([]string, 0, len(parts))
	for _, p := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("corrupt assignee_ids value %q: %w", p, err)
		}
		ids = append(ids, formatUserID(id))
	}
	return ids, nil
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

	assigneeIDs, err := parseAssigneeIDs(row.AssigneeIds)
	if err != nil {
		return nil, fmt.Errorf("parse assignee_ids: %w", err)
	}

	t := &apigen.Task{
		ID:          formatTaskID(row.ID),
		Title:       row.Title,
		Description: row.Description,
		Status: apigen.TaskStatus{
			Name:     row.StatusName,
			Category: apigen.StatusCategory(row.StatusCategory),
		},
		AssigneeIds: assigneeIDs,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
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

func formatUserID(id int64) string {
	return "U" + strconv.FormatInt(id, 10)
}

func parseUserID(s string) (int64, error) {
	if !strings.HasPrefix(s, "U") {
		return 0, fmt.Errorf("invalid user ID %q: must start with U", s)
	}
	id, err := strconv.ParseInt(s[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID %q: %w", s, err)
	}
	return id, nil
}

func parseTokenID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token ID %q: %w", s, err)
	}
	return id, nil
}

func userFromDB(u dbgen.User) (*apigen.User, error) {
	createdAt, err := parseTime(u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := parseTime(u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &apigen.User{
		ID:        formatUserID(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		IsOwner:   u.IsOwner != 0,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func authTokenFromDB(t dbgen.AuthToken) (*apigen.AuthToken, error) {
	createdAt, err := parseTime(t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &apigen.AuthToken{
		ID:        strconv.FormatInt(t.ID, 10),
		Name:      t.Name,
		Kind:      apigen.AuthTokenKind(t.Kind),
		CreatedAt: createdAt,
	}, nil
}

func memberFromDB(m dbgen.SpaceMember, userName, userEmail string) (*apigen.SpaceMember, error) {
	createdAt, err := parseTime(m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &apigen.SpaceMember{
		UserId:    formatUserID(m.UserID),
		UserName:  userName,
		UserEmail: userEmail,
		Role:      apigen.SpaceRole(m.Role),
		CreatedAt: createdAt,
	}, nil
}

func memberFromListRow(row dbgen.ListSpaceMembersBySpaceRow) (*apigen.SpaceMember, error) {
	createdAt, err := parseTime(row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &apigen.SpaceMember{
		UserId:    formatUserID(row.UserID),
		UserName:  row.UserName,
		UserEmail: row.UserEmail,
		Role:      apigen.SpaceRole(row.Role),
		CreatedAt: createdAt,
	}, nil
}
