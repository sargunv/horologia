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

	apigen "github.com/sargunv/tend/server/internal/api/gen"
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

	if errors.Is(err, sql.ErrNoRows) {
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

// --- Spaces ---

func (h *Handler) SpacesCreate(ctx context.Context, req *apigen.SpaceCreate) (*apigen.Space, error) {
	space, err := database.CreateSpaceWithDefaults(
		ctx, h.DB,
		req.Slug,
		req.Name,
		req.Description.Or(""),
	)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space)
}

func (h *Handler) SpacesList(ctx context.Context, params apigen.SpacesListParams) (*apigen.SpacePage, error) {
	cursor, err := decodeCursor(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	limit := clampLimit(params.Limit)

	q := dbgen.New(h.DB)
	spaces, err := q.ListSpaces(ctx, dbgen.ListSpacesParams{
		Slug:  cursor,
		Limit: limit + 1, // fetch one extra to determine next cursor
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(spaces, limit, spaceFromDB, func(s dbgen.Space) string {
		return s.Slug
	})
	if err != nil {
		return nil, err
	}

	return &apigen.SpacePage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) SpacesRead(ctx context.Context, params apigen.SpacesReadParams) (*apigen.Space, error) {
	q := dbgen.New(h.DB)
	space, err := q.GetSpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	return spaceFromDB(space)
}

func (h *Handler) SpacesUpdate(ctx context.Context, req *apigen.SpaceUpdate, params apigen.SpacesUpdateParams) (*apigen.Space, error) {
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

	return spaceFromDB(space)
}

func (h *Handler) SpacesDelete(ctx context.Context, params apigen.SpacesDeleteParams) error {
	q := dbgen.New(h.DB)
	result, err := q.DeleteSpace(ctx, params.SpaceSlug)
	if err != nil {
		return err
	}
	return checkDeleted(result)
}

// --- Tasks ---

func (h *Handler) SpaceTasksCreate(ctx context.Context, req *apigen.TaskCreate, params apigen.SpaceTasksCreateParams) (*apigen.Task, error) {
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
	statusName := req.StatusName.Or("")
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

	// Re-fetch with status category join.
	row, err := q.GetTaskWithStatus(ctx, task.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return taskFromDBRow(row)
}

func (h *Handler) SpaceTasksList(ctx context.Context, params apigen.SpaceTasksListParams) (*apigen.TaskPage, error) {
	cursorStr, err := decodeCursor(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	var cursorID int64
	if cursorStr != "" {
		cursorID, err = strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			return nil, badRequest("invalid cursor")
		}
	}

	limit := clampLimit(params.Limit)
	q := dbgen.New(h.DB)

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

	items, nextCursor, err := paginate(rows, limit, taskFromListRow, func(r dbgen.ListTasksBySpaceRow) string {
		return strconv.FormatInt(r.ID, 10)
	})
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPage{Items: items, NextCursor: nextCursor}, nil
}

func (h *Handler) TasksRead(ctx context.Context, params apigen.TasksReadParams) (*apigen.Task, error) {
	id, err := parseTaskID(params.TaskId)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	q := dbgen.New(h.DB)
	row, err := q.GetTaskWithStatus(ctx, id)
	if err != nil {
		return nil, err
	}
	return taskFromDBRow(row)
}

func (h *Handler) TasksUpdate(ctx context.Context, req *apigen.TaskUpdate, params apigen.TasksUpdateParams) (*apigen.Task, error) {
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
	existing, err := q.GetTaskWithStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	_, err = q.UpdateTask(ctx, dbgen.UpdateTaskParams{
		ID:          id,
		Title:       req.Title.Or(existing.Title),
		Description: req.Description.Or(existing.Description),
		StatusName:  req.StatusName.Or(existing.StatusName),
		DueDate:     dueDateFromExisting(existing.DueDate, req.DueDate),
		UpdatedAt:   now(),
	})
	if err != nil {
		return nil, err
	}

	// Re-fetch with status category join after update.
	row, err := q.GetTaskWithStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return taskFromDBRow(row)
}

func (h *Handler) TasksDelete(ctx context.Context, params apigen.TasksDeleteParams) error {
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

// NewServer creates an HTTP handler from the API spec.
func NewServer(handler apigen.Handler, log *slog.Logger) (http.Handler, error) {
	return apigen.NewServer(handler, apigen.WithErrorHandler(errorHandler(log)))
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
