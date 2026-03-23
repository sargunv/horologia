package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ogen-go/ogen/ogenerrors"

	apigen "github.com/sargunv/tend/server/api/gen"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// Handler implements the generated API interface.
type Handler struct {
	apigen.UnimplementedHandler
	DB  *sql.DB
	Log *slog.Logger
}

func (h *Handler) NewError(ctx context.Context, err error) *apigen.ApiErrorStatusCode {
	code := http.StatusInternalServerError
	apiCode := "internal_error"
	message := "an internal error occurred"

	var secErr *ogenerrors.SecurityError
	if errors.As(err, &secErr) {
		code = http.StatusUnauthorized
		apiCode = "unauthorized"
		message = "authentication required"
	} else if errors.Is(err, sql.ErrNoRows) {
		code = http.StatusNotFound
		apiCode = "not_found"
		message = "resource not found"
	} else if isUniqueViolation(err) {
		code = http.StatusConflict
		apiCode = "conflict"
		message = "resource already exists"
	} else if isForeignKeyViolation(err) {
		code = http.StatusBadRequest
		apiCode = "bad_request"
		message = "referenced resource does not exist"
	} else if isBadRequest(err) {
		code = http.StatusBadRequest
		apiCode = "bad_request"
		message = err.Error()
	}

	if code >= 500 {
		h.Log.ErrorContext(ctx, "handler error", "error", err)
	}

	return &apigen.ApiErrorStatusCode{
		StatusCode: code,
		Response: apigen.ApiError{
			Code:    apiCode,
			Message: message,
		},
	}
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func isBadRequest(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "bad request:")
}

func badRequest(msg string) error {
	return fmt.Errorf("bad request: %s", msg)
}

func checkDeleted(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

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

// --- Spaces ---

func (h *Handler) SpacesCreate(ctx context.Context, req *apigen.SpaceCreate) (*apigen.Space, error) {
	user := UserFromContext(ctx)
	space, err := database.CreateSpaceWithDefaults(
		ctx, h.DB,
		req.Slug,
		req.Name,
		req.Description.Or(""),
		user.ID,
	)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space), nil
}

func (h *Handler) SpacesList(ctx context.Context, params apigen.SpacesListParams) (*apigen.SpacePage, error) {
	user := UserFromContext(ctx)
	cursor, err := decodeCursor(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	limit := clampLimit(params.Limit)

	q := dbgen.New(h.DB)

	var spaces []dbgen.Space
	if user.IsOwner {
		spaces, err = q.ListSpaces(ctx, dbgen.ListSpacesParams{
			Slug:  cursor,
			Limit: limit + 1,
		})
	} else {
		spaces, err = q.ListSpacesByUser(ctx, dbgen.ListSpacesByUserParams{
			UserID: user.ID,
			Slug:   cursor,
			Limit:  limit + 1,
		})
	}
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(spaces, limit, func(s dbgen.Space) (*apigen.Space, error) { return spaceFromDB(s), nil }, func(s dbgen.Space) string {
		return s.Slug
	})
	if err != nil {
		return nil, err
	}

	return &apigen.SpacePage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) SpacesRead(ctx context.Context, params apigen.SpacesReadParams) (*apigen.Space, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}
	q := dbgen.New(h.DB)
	space, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space), nil
}

func (h *Handler) SpacesUpdate(ctx context.Context, req *apigen.SpaceUpdate, params apigen.SpacesUpdateParams) (*apigen.Space, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)
	existing, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	space, err := q.UpdateSpace(ctx, dbgen.UpdateSpaceParams{
		Slug:        params.SpaceSlug,
		Name:        req.Name.Or(existing.Name),
		Description: req.Description.Or(existing.Description),
		UpdatedAt:   now(),
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return spaceFromDB(space), nil
}

func (h *Handler) SpacesDelete(ctx context.Context, params apigen.SpacesDeleteParams) error {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return err
	}
	q := dbgen.New(h.DB)
	result, err := q.DeleteSpace(ctx, params.SpaceSlug)
	if err != nil {
		return err
	}
	return checkDeleted(result)
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

	ts := now()
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

	// Trim to page size and determine next cursor.
	hasMore := int64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}

	// Batch-enrich all tasks in 3 queries (not 5N).
	items, err := h.enrichTasks(ctx, q, rows)
	if err != nil {
		return nil, err
	}

	var nextCursor apigen.NilString
	if hasMore {
		nextCursor = encodeCursor(strconv.FormatInt(rows[len(rows)-1].ID, 10))
	} else {
		nextCursor.SetToNull()
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
		UpdatedAt:   now(),
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

// requireSpaceRole checks that the authenticated user has one of the given roles
// in the specified space. Global owners always pass.
func (h *Handler) requireSpaceRole(ctx context.Context, spaceSlug string, roles ...string) error {
	user := UserFromContext(ctx)
	if user.IsOwner {
		return nil
	}
	q := dbgen.New(h.DB)
	member, err := q.GetSpaceMember(ctx, dbgen.GetSpaceMemberParams{
		SpaceSlug: spaceSlug,
		UserID:    user.ID,
	})
	if err != nil {
		return err // sql.ErrNoRows -> 404 via NewError
	}
	for _, r := range roles {
		if member.Role == r {
			return nil
		}
	}
	return badRequest("insufficient permissions")
}

// requireSpaceWrite checks that the user has member or admin role.
func (h *Handler) requireSpaceWrite(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "member", "admin")
}

// requireSpaceRead checks that the user has any role in the space.
func (h *Handler) requireSpaceRead(ctx context.Context, spaceSlug string) error {
	return h.requireSpaceRole(ctx, spaceSlug, "viewer", "member", "admin")
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
	ts := now()
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

	ts := now()
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

// NewServer creates an HTTP handler from the API spec.
func NewServer(handler *Handler, log *slog.Logger) (http.Handler, error) {
	return apigen.NewServer(handler, handler, apigen.WithErrorHandler(errorHandler(log)))
}

func errorHandler(log *slog.Logger) ogenerrors.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		attrs := []any{"error", err, "method", r.Method, "path", r.URL.Path}

		var decErr *ogenerrors.DecodeRequestError
		var secErr *ogenerrors.SecurityError
		if errors.As(err, &decErr) || errors.As(err, &secErr) {
			log.DebugContext(ctx, "client error", attrs...)
		} else {
			log.ErrorContext(ctx, "server error", attrs...)
		}

		ogenerrors.DefaultErrorHandler(ctx, w, r, err)
	}
}
