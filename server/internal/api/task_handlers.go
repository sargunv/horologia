package api

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/taskengine"
	"github.com/sargunv/tend/server/internal/types"
)

// fetchTaskRelations fetches all relations for a task from both directions.
func (h *Handler) fetchTaskRelations(ctx context.Context, q *dbgen.Queries, id int64, spaceSlug string) ([]taskRelationRow, error) {
	asSource, err := q.ListRelationsByTaskAsSource(ctx, dbgen.ListRelationsByTaskAsSourceParams{
		SourceTaskID: id,
		SpaceSlug:    spaceSlug,
	})
	if err != nil {
		return nil, err
	}
	asTarget, err := q.ListRelationsByTaskAsTarget(ctx, dbgen.ListRelationsByTaskAsTargetParams{
		TargetTaskID: id,
		SpaceSlug:    spaceSlug,
	})
	if err != nil {
		return nil, err
	}
	rows := make([]taskRelationRow, 0, len(asSource)+len(asTarget))
	for _, r := range asSource {
		rows = append(rows, taskRelationRow(r))
	}
	for _, r := range asTarget {
		rows = append(rows, taskRelationRow(r))
	}
	return rows, nil
}

// fetchTask fetches a task by ID within a space, along with its assignees,
// tags, relations, and rotation pool.
func (h *Handler) fetchTask(ctx context.Context, q *dbgen.Queries, id int64, spaceSlug string) (*apigen.Task, error) {
	task, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: id, SpaceSlug: spaceSlug})
	if err != nil {
		return nil, err
	}
	assigneeIDs, err := q.ListAssigneeUserIDsByTask(ctx, id)
	if err != nil {
		return nil, err
	}
	tagNames, err := q.ListTagNamesByTask(ctx, id)
	if err != nil {
		return nil, err
	}
	relations, err := h.fetchTaskRelations(ctx, q, id, spaceSlug)
	if err != nil {
		return nil, err
	}
	poolUserIDs, err := q.ListRotationPoolByTask(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskFromDB(task, assigneeIDs, tagNames, relations, poolUserIDs)
}

// enrichTasks batch-fetches assignees, tags, relations, and rotation pool for a
// slice of tasks and converts them to API types. Uses 4 queries total instead of N*5.
func (h *Handler) enrichTasks(ctx context.Context, q *dbgen.Queries, spaceSlug string, tasks []dbgen.Task) ([]apigen.Task, error) {
	if len(tasks) == 0 {
		return []apigen.Task{}, nil
	}

	// Collect task IDs for batch queries.
	taskIDs := make([]int64, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}

	// Batch-fetch assignees, tags, relations, and rotation pool (4 queries total).
	assigneeRows, err := q.ListAssigneeUserIDsByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	tagRows, err := q.ListTagNamesByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	relationRows, err := q.ListRelationsByTasks(ctx, dbgen.ListRelationsByTasksParams{
		SpaceSlug:     spaceSlug,
		SourceTaskIds: taskIDs,
		TargetTaskIds: taskIDs,
	})
	if err != nil {
		return nil, err
	}
	poolRows, err := q.ListRotationPoolByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}

	// Group assignees by task ID.
	assigneeMap := make(map[int64][]int64)
	for _, r := range assigneeRows {
		assigneeMap[r.TaskID] = append(assigneeMap[r.TaskID], r.UserID)
	}

	// Group tag names by task ID.
	tagMap := make(map[int64][]string)
	for _, r := range tagRows {
		tagMap[r.TaskID] = append(tagMap[r.TaskID], r.Name)
	}

	// Group relations by task ID (a relation appears for both source and target).
	relationMap := make(map[int64][]taskRelationRow)
	for _, r := range relationRows {
		row := taskRelationRow(r)
		if r.SourceTaskID != r.TargetTaskID {
			relationMap[r.SourceTaskID] = append(relationMap[r.SourceTaskID], row)
			relationMap[r.TargetTaskID] = append(relationMap[r.TargetTaskID], row)
		}
	}

	// Group rotation pool by task ID. Order preserved by SQL ORDER BY position ASC.
	poolMap := make(map[int64][]int64)
	for _, r := range poolRows {
		poolMap[r.TaskID] = append(poolMap[r.TaskID], r.UserID)
	}

	// Convert each task with its pre-fetched related data.
	result := make([]apigen.Task, 0, len(tasks))
	for _, task := range tasks {
		assignees := assigneeMap[task.ID]
		if assignees == nil {
			assignees = []int64{}
		}
		tags := tagMap[task.ID]
		if tags == nil {
			tags = []string{}
		}
		relations := relationMap[task.ID]
		if relations == nil {
			relations = []taskRelationRow{}
		}
		pool := poolMap[task.ID]
		if pool == nil {
			pool = []int64{}
		}
		apiTask, err := taskFromDB(task, assignees, tags, relations, pool)
		if err != nil {
			return nil, err
		}
		result = append(result, *apiTask)
	}
	return result, nil
}

// --- Tasks ---

func (h *Handler) SpaceTasksCreate(ctx context.Context, req *apigen.TaskCreate, params apigen.SpaceTasksCreateParams) (*apigen.Task, error) {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Resolve the status name.
	statusName := req.Status.Or("")
	if statusName == "" {
		var err error
		statusName, err = taskengine.FindInitialStatus(ctx, tx, params.SpaceSlug)
		if err != nil {
			return nil, err
		}
	}

	effortName := optStringToDB(req.Effort)
	if err := validateLevel(ctx, effortName, "effort level", func(ctx context.Context) ([]dbgen.TaskEffortLevel, error) {
		return q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	}, func(l dbgen.TaskEffortLevel) string { return l.Name }); err != nil {
		return nil, err
	}

	priorityName := optStringToDB(req.Priority)
	if err := validateLevel(ctx, priorityName, "priority level", func(ctx context.Context) ([]dbgen.TaskPriorityLevel, error) {
		return q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	}, func(l dbgen.TaskPriorityLevel) string { return l.Name }); err != nil {
		return nil, err
	}

	recurrenceType := dbgen.RecurrenceType(req.RecurrenceType.Or(apigen.TaskRecurrenceTypeOneOff))
	recurrenceRule := optStringToDB(req.RecurrenceRule)

	due, err := dueToDB(req.Due)
	if err != nil {
		return nil, err
	}

	ts := time.Now()
	if err := taskengine.ValidateRecurrence(recurrenceType, recurrenceRule, ts); err != nil {
		return nil, err
	}
	dueAt, dueTz := types.DecomposeDueDate(due)
	tstz := timeToTS(ts)
	task, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		SpaceSlug:      params.SpaceSlug,
		Title:          req.Title,
		Description:    req.Description.Or(""),
		StatusName:     statusName,
		EffortName:     effortName,
		PriorityName:   priorityName,
		DueAt:          dueAt,
		DueTz:          dueTz,
		RecurrenceType: recurrenceType,
		RecurrenceRule: recurrenceRule,
		CreatedAt:      tstz,
		UpdatedAt:      tstz,
	})
	if err != nil {
		return nil, err
	}

	if err := h.applyTaskCollections(ctx, q, task.ID, params.SpaceSlug, req.AssigneeIds, req.RotationPool, req.Tags); err != nil {
		return nil, err
	}

	// Re-fetch with assignees, tags, relations, and rotation pool.
	result, err := h.fetchTask(ctx, q, task.ID, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (h *Handler) SpaceTasksList(ctx context.Context, params apigen.SpaceTasksListParams) (*apigen.TaskPage, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)

	q := dbgen.New(h.Pool)

	rows, err := q.ListTasksBySpace(ctx, dbgen.ListTasksBySpaceParams{
		SpaceSlug: params.SpaceSlug,
		ID:        cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(rows, limit, func(rows []dbgen.Task) ([]apigen.Task, error) {
		return h.enrichTasks(ctx, q, params.SpaceSlug, rows)
	}, func(t dbgen.Task) string {
		return strconv.FormatInt(t.ID, 10)
	})
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) SpaceTasksRead(ctx context.Context, params apigen.SpaceTasksReadParams) (*apigen.Task, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	id, err := parseTaskID(params.TaskId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	q := dbgen.New(h.Pool)
	return h.fetchTask(ctx, q, id, params.SpaceSlug)
}

func (h *Handler) SpaceTasksUpdate(ctx context.Context, req *apigen.TaskUpdate, params apigen.SpaceTasksUpdateParams) (*apigen.Task, error) {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	id, err := parseTaskID(params.TaskId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return nil, err
	}

	effortName := optNilStringToDB(req.Effort, existing.EffortName)
	if err := validateLevel(ctx, effortName, "effort level", func(ctx context.Context) ([]dbgen.TaskEffortLevel, error) {
		return q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	}, func(l dbgen.TaskEffortLevel) string { return l.Name }); err != nil {
		return nil, err
	}

	priorityName := optNilStringToDB(req.Priority, existing.PriorityName)
	if err := validateLevel(ctx, priorityName, "priority level", func(ctx context.Context) ([]dbgen.TaskPriorityLevel, error) {
		return q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	}, func(l dbgen.TaskPriorityLevel) string { return l.Name }); err != nil {
		return nil, err
	}

	// Merge recurrence fields. Auto-clear rule when switching to a no-rule type.
	recurrenceType := dbgen.RecurrenceType(req.RecurrenceType.Or(apigen.TaskRecurrenceType(existing.RecurrenceType)))
	recurrenceRule := optNilStringToDB(req.RecurrenceRule, existing.RecurrenceRule)
	if recurrenceType == dbgen.RecurrenceTypeOneOff || recurrenceType == dbgen.RecurrenceTypeOnDependency {
		recurrenceRule = pgtype.Text{}
	}
	now := time.Now()
	if err := taskengine.ValidateRecurrence(recurrenceType, recurrenceRule, now); err != nil {
		return nil, err
	}

	newStatus := req.Status.Or(existing.StatusName)
	newDue, err := dueFromExisting(types.NewDueDate(existing.DueAt, existing.DueTz), req.Due)
	if err != nil {
		return nil, err
	}

	cr, err := taskengine.HandleCompletionTransition(
		ctx, tx, existing, newStatus,
		recurrenceType, recurrenceRule,
		newDue, existing.LastCompletedAt,
		params.SpaceSlug, id, now,
	)
	if err != nil {
		return nil, err
	}

	crDueAt, crDueTz := types.DecomposeDueDate(cr.Due)
	_, err = q.UpdateTask(ctx, dbgen.UpdateTaskParams{
		Title:           req.Title.Or(existing.Title),
		Description:     req.Description.Or(existing.Description),
		StatusName:      cr.Status,
		EffortName:      effortName,
		PriorityName:    priorityName,
		DueAt:           crDueAt,
		DueTz:           crDueTz,
		RecurrenceType:  cr.RecurrenceType,
		RecurrenceRule:  cr.RecurrenceRule,
		LastCompletedAt: cr.LastCompletedAt,
		UpdatedAt:       timeToTS(now),
		ID:              id,
		SpaceSlug:       params.SpaceSlug,
	})
	if err != nil {
		return nil, err
	}

	// Trigger dependents when a task is completed.
	if cr.JustCompleted {
		if err := taskengine.ApplyCompletionTriggers(ctx, tx, id, params.SpaceSlug, timeToTS(now)); err != nil {
			return nil, err
		}
	}

	if err := h.applyTaskCollections(ctx, q, id, params.SpaceSlug, req.AssigneeIds, req.RotationPool, req.Tags); err != nil {
		return nil, err
	}

	// Re-fetch with assignees, tags, relations, and rotation pool.
	result, err := h.fetchTask(ctx, q, id, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (h *Handler) SpaceTasksDelete(ctx context.Context, params apigen.SpaceTasksDeleteParams) error {
	if err := h.requireSpaceWrite(ctx, params.SpaceSlug); err != nil {
		return err
	}
	id, err := parseTaskID(params.TaskId)
	if err != nil {
		return badRequest(err.Error())
	}
	q := dbgen.New(h.Pool)
	result, err := q.DeleteTask(ctx, dbgen.DeleteTaskParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}

// applyTaskCollections replaces assignees, rotation pool, and tags for a task
// when the corresponding slices are non-nil. It pre-fetches the member set once
// if both assignees and rotation pool are provided.
func (h *Handler) applyTaskCollections(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, assigneeIDs, poolIDs, tagNames []string) error {
	var memberSet map[int64]struct{}
	var err error
	if len(assigneeIDs) > 0 && len(poolIDs) > 0 {
		memberSet, err = fetchMemberSet(ctx, q, spaceSlug)
		if err != nil {
			return err
		}
	}
	if assigneeIDs != nil {
		if err := h.setTaskAssignees(ctx, q, taskID, spaceSlug, assigneeIDs, memberSet); err != nil {
			return err
		}
	}
	if poolIDs != nil {
		if err := h.setTaskRotationPool(ctx, q, taskID, spaceSlug, poolIDs, memberSet); err != nil {
			return err
		}
	}
	if tagNames != nil {
		if err := h.setTaskTags(ctx, q, taskID, spaceSlug, tagNames); err != nil {
			return err
		}
	}
	return nil
}

// fetchMemberSet returns the set of user IDs that are members of the space.
func fetchMemberSet(ctx context.Context, q *dbgen.Queries, spaceSlug string) (map[int64]struct{}, error) {
	memberIDs, err := q.ListSpaceMemberUserIDs(ctx, spaceSlug)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[int64]struct{}, len(memberIDs))
	for _, mid := range memberIDs {
		memberSet[mid] = struct{}{}
	}
	return memberSet, nil
}

// parseAndValidateUserIDs parses string user IDs, deduplicates them (preserving
// order), and validates that each user is a member of the space. If memberSet
// is nil, the member list is fetched from the database.
func parseAndValidateUserIDs(ctx context.Context, q *dbgen.Queries, spaceSlug string, rawIDs []string, memberSet map[int64]struct{}) ([]int64, error) {
	seen := make(map[int64]struct{}, len(rawIDs))
	userIDs := make([]int64, 0, len(rawIDs))
	for _, raw := range rawIDs {
		uid, err := parseUserID(raw)
		if err != nil {
			return nil, badRequest(err.Error())
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		userIDs = append(userIDs, uid)
	}

	if memberSet == nil && len(userIDs) > 0 {
		var err error
		memberSet, err = fetchMemberSet(ctx, q, spaceSlug)
		if err != nil {
			return nil, err
		}
	}
	for _, uid := range userIDs {
		if _, ok := memberSet[uid]; !ok {
			return nil, badRequest(fmt.Sprintf("user %s is not a member of this space", formatUserID(uid)))
		}
	}
	return userIDs, nil
}

// setTaskAssignees replaces all assignees for a task. It validates that each
// user is a member of the task's space. If memberSet is non-nil it is reused
// instead of re-querying the database.
// The caller must pass a transactional *dbgen.Queries to ensure atomicity.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskAssignees(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, assigneeIDs []string, memberSet map[int64]struct{}) error {
	userIDs, err := parseAndValidateUserIDs(ctx, q, spaceSlug, assigneeIDs, memberSet)
	if err != nil {
		return err
	}
	if err := q.DeleteTaskAssignees(ctx, taskID); err != nil {
		return err
	}
	tstz := timeToTS(time.Now())
	for _, uid := range userIDs {
		if err := q.InsertTaskAssignee(ctx, dbgen.InsertTaskAssigneeParams{
			TaskID:    taskID,
			UserID:    uid,
			CreatedAt: tstz,
		}); err != nil {
			return err
		}
	}
	return nil
}

// setTaskRotationPool replaces the rotation pool for a task. It validates that
// each user is a member of the task's space. Order is preserved via position.
// If memberSet is non-nil it is reused instead of re-querying the database.
// The caller must pass a transactional *dbgen.Queries to ensure atomicity.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskRotationPool(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, poolIDs []string, memberSet map[int64]struct{}) error {
	userIDs, err := parseAndValidateUserIDs(ctx, q, spaceSlug, poolIDs, memberSet)
	if err != nil {
		return err
	}
	if err := q.DeleteRotationPool(ctx, taskID); err != nil {
		return err
	}
	tstz := timeToTS(time.Now())
	for i, uid := range userIDs {
		if err := q.InsertRotationPoolMember(ctx, dbgen.InsertRotationPoolMemberParams{
			TaskID:    taskID,
			UserID:    uid,
			Position:  int32(i),
			CreatedAt: tstz,
		}); err != nil {
			return err
		}
	}
	return nil
}

// setTaskTags replaces all tags for a task. Unknown tag names are auto-created
// in the task's space. The caller must pass a transactional *dbgen.Queries.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskTags(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, tagNames []string) error {
	// Delete existing tags.
	if err := q.DeleteTaskTags(ctx, taskID); err != nil {
		return err
	}

	if len(tagNames) == 0 {
		return nil
	}

	// Deduplicate by folded name, preserving first occurrence's display name.
	type tagEntry struct {
		displayName string
		foldedName  string
	}
	seen := make(map[string]struct{}, len(tagNames))
	entries := make([]tagEntry, 0, len(tagNames))
	for _, name := range tagNames {
		if err := validateTagName(name); err != nil {
			return err
		}
		folded := taskengine.FoldTagName(name)
		if _, ok := seen[folded]; ok {
			continue
		}
		seen[folded] = struct{}{}
		entries = append(entries, tagEntry{displayName: name, foldedName: folded})
	}

	tstz := timeToTS(time.Now())
	for _, entry := range entries {
		// Upsert the tag (no-op on conflict) and get its ID in one query.
		tag, err := q.EnsureTag(ctx, dbgen.EnsureTagParams{
			SpaceSlug:  spaceSlug,
			Name:       entry.displayName,
			NameFolded: entry.foldedName,
			CreatedAt:  tstz,
		})
		if err != nil {
			return err
		}

		if err := q.InsertTaskTag(ctx, dbgen.InsertTaskTagParams{
			TaskID:    taskID,
			TagID:     tag.ID,
			CreatedAt: tstz,
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateLevel[T any](ctx context.Context, name pgtype.Text, label string, fetch func(context.Context) ([]T, error), getName func(T) string) error {
	if !name.Valid {
		return nil
	}
	levels, err := fetch(ctx)
	if err != nil {
		return err
	}
	for _, l := range levels {
		if getName(l) == name.String {
			return nil
		}
	}
	return badRequest(fmt.Sprintf("invalid %s %q", label, name.String))
}
