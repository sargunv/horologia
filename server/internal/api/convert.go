package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
	"golang.org/x/text/cases"
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

func spaceFromDB(s dbgen.Space) *apigen.Space {
	return &apigen.Space{
		Slug:        s.Slug,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   s.CreatedAt.Time(),
		UpdatedAt:   s.UpdatedAt.Time(),
	}
}

// taskRelationRow is the common shape returned by both ListRelationsByTaskAsSource
// and ListRelationsByTaskAsTarget.
type taskRelationRow struct {
	SourceTaskID int64
	TargetTaskID int64
	Kind         string
	CreatedAt    types.EpochSeconds
}

// directedKindMap maps API-facing directed relation kinds to their stored canonical kind
// and whether source/target should be flipped.
var directedKindMap = map[apigen.TaskRelationKind]struct {
	storedKind string
	flip       bool
}{
	apigen.TaskRelationKindParentOf:  {"parent", false},
	apigen.TaskRelationKindChildOf:   {"parent", true},
	apigen.TaskRelationKindBlocks:    {"blocks", false},
	apigen.TaskRelationKindBlockedBy: {"blocks", true},
}

// inverseKindMap is derived from directedKindMap at init time. It maps
// (storedKind, isFlipped) to the API-facing directed kind.
var inverseKindMap map[struct {
	kind string
	flip bool
}]apigen.TaskRelationKind

// symmetricKinds lists stored relation kinds that are symmetric (same kind in both directions).
var symmetricKinds = map[string]bool{
	string(apigen.TaskRelationKindRelatesTo):  true,
	string(apigen.TaskRelationKindDuplicates): true,
}

func init() {
	inverseKindMap = make(map[struct {
		kind string
		flip bool
	}]apigen.TaskRelationKind, len(directedKindMap))
	for apiKind, c := range directedKindMap {
		inverseKindMap[struct {
			kind string
			flip bool
		}{c.storedKind, c.flip}] = apiKind
	}
}

func relationFromDB(rel taskRelationRow, perspectiveTaskID int64) (apigen.TaskRelation, error) {
	isSource := rel.SourceTaskID == perspectiveTaskID

	var kind apigen.TaskRelationKind
	var relatedID int64

	if isSource {
		relatedID = rel.TargetTaskID
	} else {
		relatedID = rel.SourceTaskID
	}

	key := struct {
		kind string
		flip bool
	}{rel.Kind, !isSource}

	if k, ok := inverseKindMap[key]; ok {
		kind = k
	} else if symmetricKinds[rel.Kind] {
		kind = apigen.TaskRelationKind(rel.Kind)
	} else {
		return apigen.TaskRelation{}, fmt.Errorf("unknown relation kind %q", rel.Kind)
	}

	return apigen.TaskRelation{
		Kind:      kind,
		TaskId:    formatTaskID(relatedID),
		CreatedAt: rel.CreatedAt.Time(),
	}, nil
}

func canonicalizeRelation(kind apigen.TaskRelationKind, sourceID, targetID int64) (storedKind string, storedSource, storedTarget int64) {
	if c, ok := directedKindMap[kind]; ok {
		if c.flip {
			sourceID, targetID = targetID, sourceID
		}
		return c.storedKind, sourceID, targetID
	}
	// Symmetric kinds: normalize order so the lower ID is always source.
	return string(kind), min(sourceID, targetID), max(sourceID, targetID)
}

func taskFromDB(task dbgen.Task, assigneeUserIDs []int64, tagNames []string, relations []taskRelationRow) (*apigen.Task, error) {
	assigneeIDs := make([]string, len(assigneeUserIDs))
	for i, uid := range assigneeUserIDs {
		assigneeIDs[i] = formatUserID(uid)
	}

	apiRelations := make([]apigen.TaskRelation, 0, len(relations))
	for _, rel := range relations {
		r, err := relationFromDB(rel, task.ID)
		if err != nil {
			return nil, err
		}
		apiRelations = append(apiRelations, r)
	}

	t := &apigen.Task{
		ID:          formatTaskID(task.ID),
		Title:       task.Title,
		Description: task.Description,
		Status:      task.StatusName,
		AssigneeIds: assigneeIDs,
		Tags:        tagNames,
		Relations:   apiRelations,
		CreatedAt:   task.CreatedAt.Time(),
		UpdatedAt:   task.UpdatedAt.Time(),
	}

	if task.DueDate != nil {
		d, err := time.Parse("2006-01-02", *task.DueDate)
		if err != nil {
			return nil, fmt.Errorf("parse due_date: %w", err)
		}
		t.DueDate.SetTo(d)
	} else {
		t.DueDate.SetToNull()
	}

	return t, nil
}

// convertEach wraps a per-element converter into the slice converter
// signature expected by paginate.
func convertEach[DB any, API any](f func(DB) *API) func([]DB) ([]API, error) {
	return func(rows []DB) ([]API, error) {
		items := make([]API, len(rows))
		for i, r := range rows {
			items[i] = *f(r)
		}
		return items, nil
	}
}

func paginate[DB any, API any](
	rows []DB,
	limit int64,
	convertAll func([]DB) ([]API, error),
	cursorOf func(DB) string,
) ([]API, apigen.NilString, error) {
	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}
	items, err := convertAll(rows)
	if err != nil {
		return nil, apigen.NilString{}, err
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
	s := opt.Value.Format("2006-01-02")
	return &s
}

func dueDateFromExisting(existing *string, update apigen.OptNilDate) *string {
	if !update.IsSet() {
		return existing
	}
	if update.IsNull() {
		return nil
	}
	s := update.Value.Format("2006-01-02")
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

func userFromDB(u dbgen.User) *apigen.User {
	return &apigen.User{
		ID:        formatUserID(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		IsOwner:   u.IsOwner != 0,
		CreatedAt: u.CreatedAt.Time(),
		UpdatedAt: u.UpdatedAt.Time(),
	}
}

func authTokenFromDB(t dbgen.AuthToken) *apigen.AuthToken {
	return &apigen.AuthToken{
		ID:        strconv.FormatInt(t.ID, 10),
		Name:      t.Name,
		Kind:      apigen.AuthTokenKind(t.Kind),
		CreatedAt: t.CreatedAt.Time(),
	}
}

func memberToAPI(userID int64, userName, userEmail, role string, createdAt types.EpochSeconds) *apigen.SpaceMember {
	return &apigen.SpaceMember{
		UserId:    formatUserID(userID),
		UserName:  userName,
		UserEmail: userEmail,
		Role:      apigen.SpaceRole(role),
		CreatedAt: createdAt.Time(),
	}
}

var caseFolder = cases.Fold(cases.HandleFinalSigma(false))

func foldTagName(name string) string {
	return caseFolder.String(name)
}

func validateTagName(name string) error {
	if name == "" {
		return badRequest("tag name cannot be empty")
	}
	return nil
}

func tagFromDB(t dbgen.Tag) *apigen.Tag {
	return &apigen.Tag{
		Name:      t.Name,
		CreatedAt: t.CreatedAt.Time(),
	}
}
