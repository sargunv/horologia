package api

import (
	"context"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) UsersMe(ctx context.Context) (*apigen.User, error) {
	authUser := auth.UserFromContext(ctx)
	q := dbgen.New(h.Pool)
	user, err := q.GetUserByID(ctx, authUser.ID)
	if err != nil {
		return nil, err
	}
	return userFromDB(user), nil
}

func (h *Handler) UserTasksList(ctx context.Context, params apigen.UserTasksListParams) (*apigen.TaskPage, error) {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return nil, forbidden("authentication required")
	}

	requestedID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	if user.ID != requestedID && !user.IsOwner {
		return nil, forbidden("can only view your own tasks")
	}

	cursor, err := decodeTaskListCursor(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)

	q := dbgen.New(h.Pool)

	rows, err := q.ListTasksByUser(ctx, dbgen.ListTasksByUserParams{
		AssigneeUserID:     requestedID,
		CursorSortStatus:   cursor.SortStatus,
		CursorSortDue:      cursor.SortDue,
		CursorSortPriority: cursor.SortPriority,
		CursorSortEffort:   cursor.SortEffort,
		CursorID:           cursor.ID,
		Lim:                limit + 1,
	})
	if err != nil {
		return nil, err
	}

	items, nextCursor, err := paginate(rows, limit,
		func(rows []dbgen.ListTasksByUserRow) ([]apigen.Task, error) {
			tasks := make([]dbgen.Task, len(rows))
			for i, r := range rows {
				tasks[i] = taskFromUserListRow(r)
			}
			return h.enrichTasks(ctx, q, tasks)
		},
		encodeUserTaskListCursor,
	)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPage{Items: items, NextCursor: nextCursor}, nil
}
