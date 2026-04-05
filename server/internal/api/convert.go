package api

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
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

// tsToTime converts a NOT NULL pgtype.Timestamptz to time.Time.
// It panics if ts is not valid, indicating a bug (nullable column passed to a NOT NULL helper).
func tsToTime(ts pgtype.Timestamptz) time.Time {
	if !ts.Valid {
		panic("tsToTime called with invalid (NULL) Timestamptz; use only for NOT NULL columns")
	}
	return ts.Time
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

// directedKindMap maps API-facing directed relation kinds to their stored canonical kind
// and whether source/target should be flipped.
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

// inverseKindMap is derived from directedKindMap. It maps
// (storedKind, isFlipped) to the API-facing directed kind.
var inverseKindMap = func() map[relationKey]apigen.TaskRelationKind {
	m := make(map[relationKey]apigen.TaskRelationKind, len(directedKindMap))
	for apiKind, c := range directedKindMap {
		m[relationKey{c.storedKind, c.flip}] = apiKind
	}
	return m
}()

// symmetricKinds lists stored relation kinds that are symmetric (same kind in both directions).
var symmetricKinds = map[dbgen.StoredRelationKind]struct{}{
	dbgen.StoredRelationKindRelatesTo:  {},
	dbgen.StoredRelationKindDuplicates: {},
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
		TaskId:    types.FormatTaskID(relatedID),
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
		assigneeIDs[i] = types.FormatUserID(uid)
	}
	poolIDs := make([]string, len(rotationPoolUserIDs))
	for i, uid := range rotationPoolUserIDs {
		poolIDs[i] = types.FormatUserID(uid)
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
		ID:             types.FormatTaskID(task.ID),
		SpaceSlug:      task.SpaceSlug,
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
			At:       due.Date.Time,
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

	t.OverdueActionRule = overdueActionRuleFromDB(task.OverdueActionAfterDays, task.OverdueAction, task.OverdueActionStatus)

	return t, nil
}

// convertAll maps a slice of DB rows to API types using the given converter.
func convertAll[DB any, API any](rows []DB, f func(DB) *API) []API {
	items := make([]API, len(rows))
	for i, r := range rows {
		items[i] = *f(r)
	}
	return items
}

// convertEach wraps a per-element converter into the slice converter
// signature expected by paginate.
func convertEach[DB any, API any](f func(DB) *API) func([]DB) ([]API, error) {
	return func(rows []DB) ([]API, error) {
		return convertAll(rows, f), nil
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
	next := apigen.NilString{Null: true}
	if hasMore {
		next = encodeCursor(cursorOf(rows[len(rows)-1]))
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
		Date: pgtype.Date{Time: opt.Value.At, Valid: true},
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

func parseTokenID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid token ID %q: %w", s, err)
	}
	return id, nil
}

func userFromDB(u dbgen.User) *apigen.User {
	return &apigen.User{
		ID:          types.FormatUserID(u.ID),
		Email:       u.Email,
		Name:        u.Name,
		IsOwner:     u.IsOwner,
		HasPassword: u.PasswordHash.Valid,
		CreatedAt:   tsToTime(u.CreatedAt),
		UpdatedAt:   tsToTime(u.UpdatedAt),
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
		UserId:    types.FormatUserID(userID),
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

// taskListCursor holds the compound keyset pagination state for task list queries.
type taskListCursor struct {
	SortStatus   int32
	SortDue      pgtype.Date
	SortPriority int32
	SortEffort   int32
	ID           int64
}

// initialTaskListCursor returns sentinel values that sort before any real row.
func initialTaskListCursor() taskListCursor {
	return taskListCursor{
		SortStatus:   math.MinInt32,
		SortDue:      pgtype.Date{InfinityModifier: pgtype.NegativeInfinity, Valid: true},
		SortPriority: math.MinInt32,
		SortEffort:   math.MinInt32,
		ID:           0,
	}
}

func formatTaskListCursor(sortStatus int32, sortDue pgtype.Date, sortPriority, sortEffort int32, id int64) string {
	// SortDue is always valid (COALESCE ensures non-NULL).
	var duePart string
	switch sortDue.InfinityModifier {
	case pgtype.Infinity:
		duePart = "inf"
	case pgtype.NegativeInfinity:
		duePart = "-inf"
	case pgtype.Finite:
		duePart = sortDue.Time.Format(time.DateOnly)
	}
	return fmt.Sprintf("%d~%s~%d~%d~%d", sortStatus, duePart, sortPriority, sortEffort, id)
}

func encodeTaskListCursor(row dbgen.ListTasksBySpaceRow) string {
	return formatTaskListCursor(row.SortStatus, row.SortDue, row.SortPriority, row.SortEffort, row.ID)
}

func encodeUserTaskListCursor(row dbgen.ListTasksByUserRow) string {
	return formatTaskListCursor(row.SortStatus, row.SortDue, row.SortPriority, row.SortEffort, row.ID)
}

func decodeTaskListCursor(opt apigen.OptString) (taskListCursor, error) {
	if !opt.IsSet() {
		return initialTaskListCursor(), nil
	}
	raw, err := decodeCursor(opt)
	if err != nil {
		return taskListCursor{}, err
	}
	if raw == "" {
		return initialTaskListCursor(), nil
	}
	parts := strings.Split(raw, "~")
	if len(parts) != 5 {
		return taskListCursor{}, fmt.Errorf("invalid cursor: expected 5 parts, got %d", len(parts))
	}

	sortStatus, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return taskListCursor{}, fmt.Errorf("invalid cursor sort_status: %w", err)
	}

	var sortDue pgtype.Date
	switch parts[1] {
	case "inf":
		sortDue = pgtype.Date{InfinityModifier: pgtype.Infinity, Valid: true}
	case "-inf":
		sortDue = pgtype.Date{InfinityModifier: pgtype.NegativeInfinity, Valid: true}
	default:
		t, err := time.Parse(time.DateOnly, parts[1])
		if err != nil {
			return taskListCursor{}, fmt.Errorf("invalid cursor sort_due: %w", err)
		}
		sortDue = pgtype.Date{Time: t, Valid: true}
	}

	sortPriority, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil {
		return taskListCursor{}, fmt.Errorf("invalid cursor sort_priority: %w", err)
	}

	sortEffort, err := strconv.ParseInt(parts[3], 10, 32)
	if err != nil {
		return taskListCursor{}, fmt.Errorf("invalid cursor sort_effort: %w", err)
	}

	id, err := strconv.ParseInt(parts[4], 10, 64)
	if err != nil {
		return taskListCursor{}, fmt.Errorf("invalid cursor id: %w", err)
	}

	return taskListCursor{
		SortStatus:   int32(sortStatus),
		SortDue:      sortDue,
		SortPriority: int32(sortPriority),
		SortEffort:   int32(sortEffort),
		ID:           id,
	}, nil
}

// taskFromListRow extracts the dbgen.Task fields from a ListTasksBySpaceRow.
func taskFromListRow(row dbgen.ListTasksBySpaceRow) dbgen.Task {
	return dbgen.Task{
		ID:              row.ID,
		SpaceSlug:       row.SpaceSlug,
		Title:           row.Title,
		Description:     row.Description,
		StatusName:      row.StatusName,
		EffortName:      row.EffortName,
		PriorityName:    row.PriorityName,
		DueAt:           row.DueAt,
		DueTz:           row.DueTz,
		RecurrenceType:  row.RecurrenceType,
		RecurrenceRule:  row.RecurrenceRule,
		LastCompletedAt: row.LastCompletedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

// taskFromUserListRow extracts the dbgen.Task fields from a ListTasksByUserRow.
func taskFromUserListRow(row dbgen.ListTasksByUserRow) dbgen.Task {
	return dbgen.Task{
		ID:              row.ID,
		SpaceSlug:       row.SpaceSlug,
		Title:           row.Title,
		Description:     row.Description,
		StatusName:      row.StatusName,
		EffortName:      row.EffortName,
		PriorityName:    row.PriorityName,
		DueAt:           row.DueAt,
		DueTz:           row.DueTz,
		RecurrenceType:  row.RecurrenceType,
		RecurrenceRule:  row.RecurrenceRule,
		LastCompletedAt: row.LastCompletedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func nilStringFromDB(s pgtype.Text) apigen.NilString {
	if s.Valid {
		return apigen.NewNilString(s.String)
	}
	return apigen.NilString{Null: true}
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

// overdueActionRuleToDB converts an API TaskOverdueActionRule to DB columns.
func overdueActionRuleToDB(rule apigen.TaskOverdueActionRule) (afterDays pgtype.Int4, action dbgen.NullOverdueAction, statusName pgtype.Text) {
	if rule.After.IsNull() {
		afterDays = pgtype.Int4{}
	} else {
		afterDays = pgtype.Int4{Int32: rule.After.Value, Valid: true}
	}
	action = dbgen.NullOverdueAction{OverdueAction: dbgen.OverdueAction(rule.Action), Valid: true}
	if rule.Status.IsSet() {
		statusName = pgtype.Text{String: rule.Status.Value, Valid: true}
	}
	return afterDays, action, statusName
}

// overdueActionRuleFromDB converts DB overdue action columns to an API NilTaskOverdueActionRule.
func overdueActionRuleFromDB(afterDays pgtype.Int4, action dbgen.NullOverdueAction, statusName pgtype.Text) apigen.NilTaskOverdueActionRule {
	if !action.Valid {
		var nilRule apigen.NilTaskOverdueActionRule
		nilRule.SetToNull()
		return nilRule
	}
	rule := apigen.TaskOverdueActionRule{
		Action: apigen.TaskOverdueAction(action.OverdueAction),
	}
	if afterDays.Valid {
		rule.After.SetTo(afterDays.Int32)
	} else {
		rule.After.SetToNull()
	}
	if statusName.Valid {
		rule.Status.SetTo(statusName.String)
	}
	return apigen.NewNilTaskOverdueActionRule(rule)
}

// parseOptNilOverdueActionRule parses an OptNilTaskOverdueActionRule (from create/update requests)
// into DB columns. If not set, returns the existing values. If null, clears them.
func parseOptNilOverdueActionRule(
	opt apigen.OptNilTaskOverdueActionRule,
	existingAfterDays pgtype.Int4,
	existingAction dbgen.NullOverdueAction,
	existingStatus pgtype.Text,
) (afterDays pgtype.Int4, action dbgen.NullOverdueAction, statusName pgtype.Text) {
	if !opt.IsSet() {
		return existingAfterDays, existingAction, existingStatus
	}
	if opt.IsNull() {
		return pgtype.Int4{}, dbgen.NullOverdueAction{}, pgtype.Text{}
	}
	a, b, c := overdueActionRuleToDB(opt.Value)
	return a, b, c
}
