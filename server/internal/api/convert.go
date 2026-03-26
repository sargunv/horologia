package api

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

func clampLimit(opt apigen.OptInt32) int32 {
	limit := int32(defaultLimit)
	if opt.IsSet() {
		limit = opt.Value
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
		return 0, fmt.Errorf("invalid cursor: %w", err)
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

func tsToTime(ts pgtype.Timestamptz) time.Time {
	return ts.Time
}

func timeToTS(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func spaceFromDB(s dbgen.Space) *apigen.Space {
	return &apigen.Space{
		Slug:        s.Slug,
		Name:        s.Name,
		Description: s.Description,
		CreatedAt:   tsToTime(s.CreatedAt),
		UpdatedAt:   tsToTime(s.UpdatedAt),
	}
}

// taskRelationRow is the common shape returned by both ListRelationsByTaskAsSource
// and ListRelationsByTaskAsTarget.
type taskRelationRow struct {
	SourceTaskID int64
	TargetTaskID int64
	Kind         dbgen.StoredRelationKind
	CreatedAt    pgtype.Timestamptz
}

// directedKindMap maps API-facing directed relation kinds to their stored canonical kind,
// whether source/target should be flipped, and whether the relation should be copied
// when spawning a new task from a fixed_accumulating template.
//
// copyOnSpawn policy: true for relations that describe a task's role in a workflow
// (parent/child, blocking, triggering); false for relations specific to a particular
// instance (duplicates, spawn lineage).
var directedKindMap = map[apigen.TaskRelationKind]struct {
	storedKind dbgen.StoredRelationKind
	flip       bool
}{
	apigen.TaskRelationKindParentOf:    {dbgen.StoredRelationKindParent, false},
	apigen.TaskRelationKindChildOf:     {dbgen.StoredRelationKindParent, true},
	apigen.TaskRelationKindBlocks:      {dbgen.StoredRelationKindBlocks, false},
	apigen.TaskRelationKindBlockedBy:   {dbgen.StoredRelationKindBlocks, true},
	apigen.TaskRelationKindTriggers:    {dbgen.StoredRelationKindTriggers, false},
	apigen.TaskRelationKindTriggeredBy: {dbgen.StoredRelationKindTriggers, true},
	apigen.TaskRelationKindSpawns:      {dbgen.StoredRelationKindSpawns, false},
	apigen.TaskRelationKindSpawnedBy:   {dbgen.StoredRelationKindSpawns, true},
}

// relationKey identifies a stored relation by its canonical kind and direction.
type relationKey struct {
	kind dbgen.StoredRelationKind
	flip bool
}

// inverseKindMap is derived from directedKindMap at init time. It maps
// (storedKind, isFlipped) to the API-facing directed kind.
var inverseKindMap map[relationKey]apigen.TaskRelationKind

// symmetricKinds lists stored relation kinds that are symmetric (same kind in both directions).
var symmetricKinds = map[dbgen.StoredRelationKind]struct{}{
	dbgen.StoredRelationKindRelatesTo:  {},
	dbgen.StoredRelationKindDuplicates: {},
}

func init() {
	inverseKindMap = make(map[relationKey]apigen.TaskRelationKind, len(directedKindMap))
	for apiKind, c := range directedKindMap {
		inverseKindMap[relationKey{c.storedKind, c.flip}] = apiKind
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

	key := relationKey{rel.Kind, !isSource}

	if k, ok := inverseKindMap[key]; ok {
		kind = k
	} else if _, ok := symmetricKinds[rel.Kind]; ok {
		kind = apigen.TaskRelationKind(rel.Kind)
	} else {
		return apigen.TaskRelation{}, fmt.Errorf("unknown relation kind %q", rel.Kind)
	}

	return apigen.TaskRelation{
		Kind:      kind,
		TaskId:    formatTaskID(relatedID),
		CreatedAt: tsToTime(rel.CreatedAt),
	}, nil
}

func canonicalizeRelation(kind apigen.TaskRelationKind, sourceID, targetID int64) (storedKind dbgen.StoredRelationKind, storedSource, storedTarget int64, err error) {
	if c, ok := directedKindMap[kind]; ok {
		if c.flip {
			sourceID, targetID = targetID, sourceID
		}
		return c.storedKind, sourceID, targetID, nil
	}
	// Symmetric kinds: normalize order so the lower ID is always source.
	sk := dbgen.StoredRelationKind(kind)
	if _, ok := symmetricKinds[sk]; !ok {
		return "", 0, 0, fmt.Errorf("unknown relation kind %q", kind)
	}
	return sk, min(sourceID, targetID), max(sourceID, targetID), nil
}

func taskFromDB(task dbgen.Task, assigneeUserIDs []int64, tagNames []string, relations []taskRelationRow, rotationPoolUserIDs []int64) (*apigen.Task, error) {
	assigneeIDs := make([]string, len(assigneeUserIDs))
	for i, uid := range assigneeUserIDs {
		assigneeIDs[i] = formatUserID(uid)
	}
	poolIDs := make([]string, len(rotationPoolUserIDs))
	for i, uid := range rotationPoolUserIDs {
		poolIDs[i] = formatUserID(uid)
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
		ID:             formatTaskID(task.ID),
		Title:          task.Title,
		Description:    task.Description,
		Status:         task.StatusName,
		Effort:         nilStringFromDB(task.EffortName),
		Priority:       nilStringFromDB(task.PriorityName),
		RecurrenceType: apigen.TaskRecurrenceType(task.RecurrenceType),
		RecurrenceRule: nilStringFromDB(task.RecurrenceRule),
		AssigneeIds:    assigneeIDs,
		RotationPool:   poolIDs,
		Tags:           tagNames,
		Relations:      apiRelations,
		CreatedAt:      tsToTime(task.CreatedAt),
		UpdatedAt:      tsToTime(task.UpdatedAt),
	}

	if due := types.NewDueDate(task.DueAt, task.DueTz); due != nil {
		t.Due.SetTo(apigen.TaskDue{
			At:       due.Date,
			Timezone: due.Tz,
		})
	} else {
		t.Due.SetToNull()
	}

	if task.LastCompletedAt.Valid {
		t.LastCompletedAt.SetTo(task.LastCompletedAt.Time)
	} else {
		t.LastCompletedAt.SetToNull()
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
	limit int32,
	convertAll func([]DB) ([]API, error),
	cursorOf func(DB) string,
) ([]API, apigen.NilString, error) {
	hasMore := len(rows) > int(limit)
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

// dueToDB extracts a DueDate from an OptNilTaskDue (for create/update).
// Returns nil if not set or null. Validates the timezone.
func dueToDB(opt apigen.OptNilTaskDue) (*types.DueDate, error) {
	if !opt.IsSet() || opt.IsNull() {
		return nil, nil
	}
	if _, err := time.LoadLocation(opt.Value.Timezone); err != nil {
		return nil, badRequest(fmt.Sprintf("invalid timezone %q", opt.Value.Timezone))
	}
	return &types.DueDate{
		Date: opt.Value.At,
		Tz:   opt.Value.Timezone,
	}, nil
}

// dueFromExisting merges the due field from a PATCH request with existing values.
// Absent = no change, null = clear, object = set both.
func dueFromExisting(existing *types.DueDate, update apigen.OptNilTaskDue) (*types.DueDate, error) {
	if !update.IsSet() {
		return existing, nil
	}
	return dueToDB(update)
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
		IsOwner:   u.IsOwner,
		CreatedAt: tsToTime(u.CreatedAt),
		UpdatedAt: tsToTime(u.UpdatedAt),
	}
}

func authTokenFromDB(t dbgen.AuthToken) *apigen.AuthToken {
	return &apigen.AuthToken{
		ID:        strconv.FormatInt(t.ID, 10),
		Name:      t.Name,
		Kind:      apigen.AuthTokenKind(t.Kind),
		CreatedAt: tsToTime(t.CreatedAt),
	}
}

func memberToAPI(userID int64, userName, userEmail string, role dbgen.SpaceRole, createdAt pgtype.Timestamptz) *apigen.SpaceMember {
	return &apigen.SpaceMember{
		UserId:    formatUserID(userID),
		UserName:  userName,
		UserEmail: userEmail,
		Role:      apigen.SpaceRole(role),
		CreatedAt: tsToTime(createdAt),
	}
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
		CreatedAt: tsToTime(t.CreatedAt),
	}
}

func statusFromDB(s dbgen.TaskStatus) *apigen.TaskStatus {
	return &apigen.TaskStatus{
		Name:     s.Name,
		Category: apigen.TaskStatusCategory(s.Category),
		Position: int64(s.Position),
	}
}

func effortLevelFromDB(e dbgen.TaskEffortLevel) *apigen.TaskEffortLevel {
	return &apigen.TaskEffortLevel{
		Name:     e.Name,
		Position: int64(e.Position),
	}
}

func priorityLevelFromDB(p dbgen.TaskPriorityLevel) *apigen.TaskPriorityLevel {
	return &apigen.TaskPriorityLevel{
		Name:     p.Name,
		Position: int64(p.Position),
	}
}

func nilStringFromDB(s pgtype.Text) apigen.NilString {
	if s.Valid {
		return apigen.NewNilString(s.String)
	}
	var ns apigen.NilString
	ns.SetToNull()
	return ns
}

func optStringToDB(opt apigen.OptString) pgtype.Text {
	if !opt.IsSet() {
		return pgtype.Text{}
	}
	return pgtype.Text{String: opt.Value, Valid: true}
}

func optNilStringToDB(opt apigen.OptNilString, existing pgtype.Text) pgtype.Text {
	if !opt.IsSet() {
		return existing
	}
	if opt.IsNull() {
		return pgtype.Text{}
	}
	return pgtype.Text{String: opt.Value, Valid: true}
}
