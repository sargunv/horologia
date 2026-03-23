package api

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

// fetchTaskRelations fetches all relations for a task from both directions.
func (h *Handler) fetchTaskRelations(ctx context.Context, q *dbgen.Queries, id int64) ([]taskRelationRow, error) {
	asSource, err := q.ListRelationsByTaskAsSource(ctx, id)
	if err != nil {
		return nil, err
	}
	asTarget, err := q.ListRelationsByTaskAsTarget(ctx, id)
	if err != nil {
		return nil, err
	}
	rows := make([]taskRelationRow, 0, len(asSource)+len(asTarget))
	for _, r := range asSource {
		rows = append(rows, taskRelationRow{
			SourceTaskID: r.SourceTaskID,
			TargetTaskID: r.TargetTaskID,
			Kind:         r.Kind,
			CreatedAt:    r.CreatedAt,
		})
	}
	for _, r := range asTarget {
		rows = append(rows, taskRelationRow{
			SourceTaskID: r.SourceTaskID,
			TargetTaskID: r.TargetTaskID,
			Kind:         r.Kind,
			CreatedAt:    r.CreatedAt,
		})
	}
	return rows, nil
}

// fetchTask fetches a task by ID along with its assignees, tags, and relations.
func (h *Handler) fetchTask(ctx context.Context, q *dbgen.Queries, id int64) (*apigen.Task, error) {
	task, err := q.GetTask(ctx, id)
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
	relations, err := h.fetchTaskRelations(ctx, q, id)
	if err != nil {
		return nil, err
	}
	return taskFromDB(task, assigneeIDs, tagNames, relations)
}

// enrichTasks batch-fetches assignees, tags, and relations for a slice of tasks
// and converts them to API types. Uses 3 queries total instead of 5N.
func (h *Handler) enrichTasks(ctx context.Context, q *dbgen.Queries, tasks []dbgen.Task) ([]apigen.Task, error) {
	if len(tasks) == 0 {
		return []apigen.Task{}, nil
	}

	// Collect task IDs for batch queries.
	taskIDs := make([]int64, len(tasks))
	for i, t := range tasks {
		taskIDs[i] = t.ID
	}

	// Batch-fetch assignees, tags, and relations (3 queries total).
	assigneeRows, err := q.ListAssigneeUserIDsByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	tagRows, err := q.ListTagNamesByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	relationRows, err := q.ListRelationsByTasks(ctx, dbgen.ListRelationsByTasksParams{
		SourceTaskIds: taskIDs,
		TargetTaskIds: taskIDs,
	})
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
		row := taskRelationRow{
			SourceTaskID: r.SourceTaskID,
			TargetTaskID: r.TargetTaskID,
			Kind:         r.Kind,
			CreatedAt:    r.CreatedAt,
		}
		if r.SourceTaskID != r.TargetTaskID {
			relationMap[r.SourceTaskID] = append(relationMap[r.SourceTaskID], row)
			relationMap[r.TargetTaskID] = append(relationMap[r.TargetTaskID], row)
		}
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
		apiTask, err := taskFromDB(task, assignees, tags, relations)
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

	// Verify the space exists.
	if _, err := q.GetSpace(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	// Resolve the status name.
	statusName := req.Status.Or("")
	if statusName == "" {
		statuses, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
		if err != nil {
			return nil, err
		}
		for _, s := range statuses {
			if s.Category == "initial" {
				statusName = s.Name
				break
			}
		}
		if statusName == "" {
			return nil, badRequest("space has no initial status")
		}
	}

	ts := types.Now()
	task, err := q.CreateTask(ctx, dbgen.CreateTaskParams{
		SpaceSlug:   params.SpaceSlug,
		Title:       req.Title,
		Description: req.Description.Or(""),
		StatusName:  statusName,
		DueDate:     dueDateToDB(req.DueDate),
		CreatedAt:   ts,
		UpdatedAt:   ts,
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

	// Set tags if provided.
	if req.Tags != nil {
		if err := h.setTaskTags(ctx, q, task.ID, params.SpaceSlug, req.Tags); err != nil {
			return nil, err
		}
	}

	// Re-fetch with assignees, tags, and relations.
	result, err := h.fetchTask(ctx, q, task.ID)
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

	// Verify the space exists.
	if _, err := q.GetSpace(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	rows, err := q.ListTasksBySpace(ctx, dbgen.ListTasksBySpaceParams{
		SpaceSlug: params.SpaceSlug,
		ID:        cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(rows, limit, func(rows []dbgen.Task) ([]apigen.Task, error) {
		return h.enrichTasks(ctx, q, rows)
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
	return h.fetchTask(ctx, q, id)
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
	existing, err := q.GetTask(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = q.UpdateTask(ctx, dbgen.UpdateTaskParams{
		ID:          id,
		Title:       req.Title.Or(existing.Title),
		Description: req.Description.Or(existing.Description),
		StatusName:  req.Status.Or(existing.StatusName),
		DueDate:     dueDateFromExisting(existing.DueDate, req.DueDate),
		UpdatedAt:   types.Now(),
	})
	if err != nil {
		return nil, err
	}

	// Replace assignees if provided (nil = no change, empty = clear all).
	if req.AssigneeIds != nil {
		if err := h.setTaskAssignees(ctx, q, id, existing.SpaceSlug, req.AssigneeIds); err != nil {
			return nil, err
		}
	}

	// Replace tags if provided (nil = no change, empty = clear all).
	if req.Tags != nil {
		if err := h.setTaskTags(ctx, q, id, existing.SpaceSlug, req.Tags); err != nil {
			return nil, err
		}
	}

	// Re-fetch with assignees, tags, and relations.
	result, err := h.fetchTask(ctx, q, id)
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
	result, err := q.DeleteTask(ctx, id)
	if err != nil {
		return err
	}
	return checkDeleted(result)
}

// setTaskAssignees replaces all assignees for a task. It validates that each
// user is a member of the task's space.
// The caller must pass a transactional *dbgen.Queries to ensure atomicity.
// Max array length is enforced by ogen's @maxItems(100) validation.
func (h *Handler) setTaskAssignees(ctx context.Context, q *dbgen.Queries, taskID int64, spaceSlug string, assigneeIDs []string) error {
	// Parse and deduplicate user IDs.
	seen := make(map[int64]struct{}, len(assigneeIDs))
	userIDs := make([]int64, 0, len(assigneeIDs))
	for _, raw := range assigneeIDs {
		uid, err := parseUserID(raw)
		if err != nil {
			return badRequest(err.Error())
		}
		if _, ok := seen[uid]; ok {
			continue
		}
		seen[uid] = struct{}{}
		userIDs = append(userIDs, uid)
	}

	// Batch-fetch space member IDs to validate membership without N+1 queries.
	memberIDs, err := q.ListSpaceMemberUserIDs(ctx, spaceSlug)
	if err != nil {
		return err
	}
	memberSet := make(map[int64]struct{}, len(memberIDs))
	for _, mid := range memberIDs {
		memberSet[mid] = struct{}{}
	}

	// Verify each user is a member of the space.
	for _, uid := range userIDs {
		if _, ok := memberSet[uid]; !ok {
			return badRequest(fmt.Sprintf("user %s is not a member of this space", formatUserID(uid)))
		}
	}

	// Delete existing assignees and insert new ones.
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
		folded := foldTagName(name)
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
