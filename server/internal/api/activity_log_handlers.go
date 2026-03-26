package api

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	apigen "github.com/sargunv/tend/server/internal/api/gen"
	"github.com/sargunv/tend/server/internal/auth"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

func (h *Handler) SpaceActivityList(ctx context.Context, params apigen.SpaceActivityListParams) (*apigen.ActivityLogPage, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)
	q := dbgen.New(h.Pool)

	rows, err := q.ListActivityLogBySpace(ctx, dbgen.ListActivityLogBySpaceParams{
		SpaceSlug: params.SpaceSlug,
		Column2:   cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	return h.paginateActivityLog(ctx, q, rows, limit)
}

func (h *Handler) SpaceTaskActivityList(ctx context.Context, params apigen.SpaceTaskActivityListParams) (*apigen.ActivityLogPage, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	if _, err := types.ParseTaskID(params.TaskId); err != nil {
		return nil, badRequest(err.Error())
	}

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)
	q := dbgen.New(h.Pool)

	rows, err := q.ListActivityLogByTask(ctx, dbgen.ListActivityLogByTaskParams{
		EntityID:  params.TaskId,
		SpaceSlug: params.SpaceSlug,
		Column3:   cursorID,
		Limit:     limit + 1,
	})
	if err != nil {
		return nil, err
	}

	return h.paginateActivityLog(ctx, q, rows, limit)
}

func (h *Handler) UserActivityList(ctx context.Context, params apigen.UserActivityListParams) (*apigen.ActivityLogPage, error) {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return nil, forbidden("authentication required")
	}

	requestedID, err := types.ParseUserID(params.UserId)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	if user.ID != requestedID && !user.IsOwner {
		return nil, forbidden("can only view your own activity")
	}

	cursorID, err := decodeCursorInt64(params.Cursor)
	if err != nil {
		return nil, badRequest(err.Error())
	}

	limit := clampLimit(params.Limit)
	q := dbgen.New(h.Pool)

	rows, err := q.ListActivityLogByActor(ctx, dbgen.ListActivityLogByActorParams{
		ActorID: pgtype.Int8{Int64: requestedID, Valid: true},
		Column2: cursorID,
		UserID:  user.ID,
		Column4: user.IsOwner,
		Limit:   limit + 1,
	})
	if err != nil {
		return nil, err
	}

	return h.paginateActivityLog(ctx, q, rows, limit)
}

// paginateActivityLog paginates a slice of activity log rows, batch-fetches
// their details, and returns the API response.
func (h *Handler) paginateActivityLog(ctx context.Context, q *dbgen.Queries, rows []dbgen.ActivityLog, limit int32) (*apigen.ActivityLogPage, error) {
	items, nextCursor, err := paginate(rows, limit, func(rows []dbgen.ActivityLog) ([]apigen.ActivityLogEntry, error) {
		return h.enrichActivityLog(ctx, q, rows)
	}, activityCursorKey)
	if err != nil {
		return nil, err
	}
	return &apigen.ActivityLogPage{Items: items, NextCursor: nextCursor}, nil
}

// enrichActivityLog batch-fetches details for a page of activity log entries.
func (h *Handler) enrichActivityLog(ctx context.Context, q *dbgen.Queries, rows []dbgen.ActivityLog) ([]apigen.ActivityLogEntry, error) {
	if len(rows) == 0 {
		return []apigen.ActivityLogEntry{}, nil
	}

	logIDs := make([]int64, len(rows))
	for i, r := range rows {
		logIDs[i] = r.ID
	}

	detailRows, err := q.ListActivityLogDetailsByLogIDs(ctx, logIDs)
	if err != nil {
		return nil, err
	}

	detailMap := make(map[int64][]apigen.ActivityDetail)
	for _, d := range detailRows {
		detail := apigen.ActivityDetail{Field: d.Field}
		if d.FromValue.Valid {
			detail.From.SetTo(d.FromValue.String)
		} else {
			detail.From.SetToNull()
		}
		if d.ToValue.Valid {
			detail.To.SetTo(d.ToValue.String)
		} else {
			detail.To.SetToNull()
		}
		detailMap[d.ActivityLogID] = append(detailMap[d.ActivityLogID], detail)
	}

	result := make([]apigen.ActivityLogEntry, len(rows))
	for i, r := range rows {
		entry := apigen.ActivityLogEntry{
			ID:         strconv.FormatInt(r.ID, 10),
			SpaceSlug:  r.SpaceSlug,
			EntityType: apigen.ActivityEntityType(r.EntityType),
			EntityId:   r.EntityID,
			Action:     apigen.ActivityAction(r.Action),
			CreatedAt:  tsToTime(r.CreatedAt),
		}

		if r.ActorID.Valid {
			entry.ActorId.SetTo(types.FormatUserID(r.ActorID.Int64))
		} else {
			entry.ActorId.SetToNull()
		}

		if r.TokenID.Valid {
			entry.TokenId.SetTo(strconv.FormatInt(r.TokenID.Int64, 10))
		} else {
			entry.TokenId.SetToNull()
		}

		if r.TokenName.Valid {
			entry.TokenName.SetTo(r.TokenName.String)
		} else {
			entry.TokenName.SetToNull()
		}

		if details, ok := detailMap[r.ID]; ok {
			entry.Details = details
		} else {
			entry.Details = []apigen.ActivityDetail{}
		}

		result[i] = entry
	}

	return result, nil
}

func activityCursorKey(e dbgen.ActivityLog) string {
	return strconv.FormatInt(e.ID, 10)
}
