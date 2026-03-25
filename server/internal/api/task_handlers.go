package api

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

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
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	// Resolve the status name.
	statusName := req.Status.Or("")
	if statusName == "" {
		var err error
		statusName, err = taskengine.FindInitialStatus(ctx, q, params.SpaceSlug)
		if err != nil {
			return nil, err
		}
	}

	effortName := optStringToDB(req.Effort)
	if err := validateOptionalLevel(ctx, q, params.SpaceSlug, effortName, "effort"); err != nil {
		return nil, err
	}

	priorityName := optStringToDB(req.Priority)
	if err := validateOptionalLevel(ctx, q, params.SpaceSlug, priorityName, "priority"); err != nil {
		return nil, err
	}

	recurrenceType := req.RecurrenceType.Or(apigen.TaskRecurrenceTypeOneOff)
	recurrenceRule := optStringToDB(req.RecurrenceRule)

	dueAt, dueTz, err := dueToDB(req.Due)
	if err != nil {
		return nil, err
	}

	ts := types.Now()
	if err := taskengine.ValidateRecurrence(recurrenceType, recurrenceRule, ts.Time()); err != nil {
		return nil, err
	}
	task, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		SpaceSlug:      params.SpaceSlug,
		Title:          req.Title,
		Description:    req.Description.Or(""),
		StatusName:     statusName,
		EffortName:     effortName,
		PriorityName:   priorityName,
		DueAt:          dueAt,
		DueTz:          dueTz,
		RecurrenceType: string(recurrenceType),
		RecurrenceRule: recurrenceRule,
		CreatedAt:      ts,
		UpdatedAt:      ts,
	})
	if err != nil {
		return nil, err
	}

	// Set assignees if provided.
	if req.AssigneeIds != nil {
		if err := h.setTaskAssignees(ctx, q, task.ID, params.SpaceSlug, req.AssigneeIds); err != nil {
			return nil, err
		}
	}

	// Set rotation pool if provided.
	if req.RotationPool != nil {
		if err := h.setTaskRotationPool(ctx, q, task.ID, params.SpaceSlug, req.RotationPool); err != nil {
			return nil, err
		}
	}

	// Set tags if provided.
	if req.Tags != nil {
		if err := h.setTaskTags(ctx, q, task.ID, params.SpaceSlug, req.Tags); err != nil {
			return nil, err
		}
	}

	// Re-fetch with assignees, tags, relations, and rotation pool.
	result, err := h.fetchTask(ctx, q, task.ID, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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

	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

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
	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)
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

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)
	existing, err := q.GetTask(ctx, dbgen.GetTaskParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return nil, err
	}

	effortName := optNilStringToDB(req.Effort, existing.EffortName)
	if err := validateOptionalLevel(ctx, q, params.SpaceSlug, effortName, "effort"); err != nil {
		return nil, err
	}

	priorityName := optNilStringToDB(req.Priority, existing.PriorityName)
	if err := validateOptionalLevel(ctx, q, params.SpaceSlug, priorityName, "priority"); err != nil {
		return nil, err
	}

	// Merge recurrence fields. Auto-clear rule when switching to a no-rule type.
	recurrenceType := req.RecurrenceType.Or(apigen.TaskRecurrenceType(existing.RecurrenceType))
	recurrenceRule := optNilStringToDB(req.RecurrenceRule, existing.RecurrenceRule)
	if recurrenceType == apigen.TaskRecurrenceTypeOneOff || recurrenceType == apigen.TaskRecurrenceTypeOnDependency {
		recurrenceRule = nil
	}
	now := types.Now()
	if err := taskengine.ValidateRecurrence(recurrenceType, recurrenceRule, now.Time()); err != nil {
		return nil, err
	}

	newStatus := req.Status.Or(existing.StatusName)
	newDueAt, newDueTz, err := dueFromExisting(existing.DueAt, existing.DueTz, req.Due)
	if err != nil {
		return nil, err
	}

	cr, err := h.Engine.HandleCompletionTransition(
		ctx, q, existing, newStatus,
		recurrenceType, recurrenceRule,
		newDueAt, newDueTz, existing.LastCompletedAt,
		params.SpaceSlug, id, now,
	)
	if err != nil {
		return nil, err
	}

	_, err = q.UpdateTask(ctx, dbgen.UpdateTaskParams{
		Title:           req.Title.Or(existing.Title),
		Description:     req.Description.Or(existing.Description),
		StatusName:      cr.Status,
		EffortName:      effortName,
		PriorityName:    priorityName,
		DueAt:           cr.DueAt,
		DueTz:           cr.DueTz,
		RecurrenceType:  string(cr.RecurrenceType),
		RecurrenceRule:  cr.RecurrenceRule,
		LastCompletedAt: cr.LastCompletedAt,
		UpdatedAt:       now,
		ID:              id,
		SpaceSlug:       params.SpaceSlug,
	})
	if err != nil {
		return nil, err
	}

	// Trigger dependents when a task is completed.
	if cr.JustCompleted {
		if err := taskengine.ApplyCompletionTriggers(ctx, q, id, params.SpaceSlug, now); err != nil {
			return nil, err
		}
	}

	// Replace assignees if provided (nil = no change, empty = clear all).
	if req.AssigneeIds != nil {
		if err := h.setTaskAssignees(ctx, q, id, params.SpaceSlug, req.AssigneeIds); err != nil {
			return nil, err
		}
	}

	// Replace rotation pool if provided (nil = no change, empty = clear all).
	if req.RotationPool != nil {
		if err := h.setTaskRotationPool(ctx, q, id, params.SpaceSlug, req.RotationPool); err != nil {
			return nil, err
		}
	}

	// Replace tags if provided (nil = no change, empty = clear all).
	if req.Tags != nil {
		if err := h.setTaskTags(ctx, q, id, params.SpaceSlug, req.Tags); err != nil {
			return nil, err
		}
	}

	// Re-fetch with assignees, tags, relations, and rotation pool.
	result, err := h.fetchTask(ctx, q, id, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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
	q := dbgen.New(h.DB)
	result, err := q.DeleteTask(ctx, dbgen.DeleteTaskParams{ID: id, SpaceSlug: params.SpaceSlug})
	if err != nil {
		return err
	}
	return checkDeleted(result)
}

// parseAndValidateUserIDs parses string user IDs, deduplicates them (preserving
// order), and validates that each user is a member of the space. Returns the
// validated int64 user IDs.
func parseAndValidateUserIDs(ctx context.Context, q *dbgen.Queries, spaceSlug string, rawIDs []string) ([]int64, error) {
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

	memberIDs, err := q.ListSpaceMemberUserIDs(ctx, spaceSlug)
	if err != nil {
		return nil, err
	}
	memberSet := make(map[int64]struct{}, len(memberIDs))
	for _, mid := range memberIDs {
		memberSet[mid] = struct{}{}
	}
	for _, uid := range userIDs {
		if _, ok := memberSet[uid]; !ok {
			return nil, badRequest(fmt.Sprintf("user %s is not a member of this space", formatUserID(uid)))
		}
	}
	return userIDs, nil
}

// setTaskAssignees replaces all assignees for a task. It validates that each
// user is a member of the task's space.
// The caller must pass a transactional *dbgen.Queries to ensure atomicity.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskAssignees(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, assigneeIDs []string) error {
	userIDs, err := parseAndValidateUserIDs(ctx, q, spaceSlug, assigneeIDs)
	if err != nil {
		return err
	}
	if err := q.DeleteTaskAssignees(ctx, taskID); err != nil {
		return err
	}
	ts := types.Now()
	for _, uid := range userIDs {
		if err := q.InsertTaskAssignee(ctx, dbgen.InsertTaskAssigneeParams{
			TaskID:    taskID,
			UserID:    uid,
			CreatedAt: ts,
		}); err != nil {
			return err
		}
	}
	return nil
}

// setTaskRotationPool replaces the rotation pool for a task. It validates that
// each user is a member of the task's space. Order is preserved via position.
// The caller must pass a transactional *dbgen.Queries to ensure atomicity.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskRotationPool(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, poolIDs []string) error {
	userIDs, err := parseAndValidateUserIDs(ctx, q, spaceSlug, poolIDs)
	if err != nil {
		return err
	}
	if err := q.DeleteRotationPool(ctx, taskID); err != nil {
		return err
	}
	ts := types.Now()
	for i, uid := range userIDs {
		if err := q.InsertRotationPoolMember(ctx, dbgen.InsertRotationPoolMemberParams{
			TaskID:    taskID,
			UserID:    uid,
			Position:  int64(i),
			CreatedAt: ts,
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

	ts := types.Now()
	for _, entry := range entries {
		// Upsert the tag (no-op on conflict) and get its ID in one query.
		tag, err := q.EnsureTag(ctx, dbgen.EnsureTagParams{
			SpaceSlug:  spaceSlug,
			Name:       entry.displayName,
			NameFolded: entry.foldedName,
			CreatedAt:  ts,
		})
		if err != nil {
			return err
		}

		if err := q.InsertTaskTag(ctx, dbgen.InsertTaskTagParams{
			TaskID:    taskID,
			TagID:     tag.ID,
			CreatedAt: ts,
		}); err != nil {
			return err
		}
	}
	return nil
}

// validateOptionalLevel checks that a non-nil level name exists in the space's
// configured levels. The label ("effort" or "priority") is used in the error message.
func validateOptionalLevel(ctx context.Context, q *dbgen.Queries, spaceSlug string, name *string, label string) error {
	if name == nil {
		return nil
	}
	var validNames []string
	switch label {
	case "effort":
		levels, err := q.ListTaskEffortLevelsBySpace(ctx, spaceSlug)
		if err != nil {
			return err
		}
		validNames = make([]string, len(levels))
		for i, l := range levels {
			validNames[i] = l.Name
		}
	case "priority":
		levels, err := q.ListTaskPriorityLevelsBySpace(ctx, spaceSlug)
		if err != nil {
			return err
		}
		validNames = make([]string, len(levels))
		for i, l := range levels {
			validNames[i] = l.Name
		}
	default:
		panic(fmt.Sprintf("validateOptionalLevel: unknown label %q", label))
	}
	for _, v := range validNames {
		if v == *name {
			return nil
		}
	}
	return badRequest(fmt.Sprintf("invalid %s level %q", label, *name))
}
