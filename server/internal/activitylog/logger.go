package activitylog

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sargunv/horologia/server/internal/auth"
	dbgen "github.com/sargunv/horologia/server/internal/database/gen"
	"github.com/sargunv/horologia/server/internal/types"
)

// EntityType identifies what kind of entity an activity log entry is about.
type EntityType dbgen.ActivityEntityType

const (
	EntityTask          EntityType = "task"
	EntityRecipe        EntityType = "recipe"
	EntitySpace         EntityType = "space"
	EntityMember        EntityType = "member"
	EntityTag           EntityType = "tag"
	EntityStatus        EntityType = "status"
	EntityEffortLevel   EntityType = "effort_level"
	EntityPriorityLevel EntityType = "priority_level"
	EntityRelation      EntityType = "relation"
	EntityUser          EntityType = "user"
)

// Action identifies what kind of action an activity log entry records.
type Action dbgen.ActivityAction

const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
)

// Detail represents a single field change or metadata entry.
type Detail struct {
	Field string
	From  *string // nil = not applicable (e.g., on create)
	To    *string // nil = not applicable (e.g., on delete)
}

// Entry describes an activity to log.
type Entry struct {
	SpaceSlug  string
	EntityType EntityType
	EntityID   string
	Action     Action
	Details    []Detail
}

// Str returns a pointer to s, for use in Detail.From/To.
//
//go:fix inline
func Str(s string) *string { return new(s) }

// Log writes an activity log entry and its details within the caller's
// transaction. It reads actor identity from the context; if absent, the
// entry is attributed as a system action (null actor_id/token_id).
func Log(ctx context.Context, db dbgen.DBTX, entry Entry, now time.Time) error {
	var actorID pgtype.Int8
	var tokenID pgtype.Int8
	var tokenName pgtype.Text

	if u := auth.UserFromContext(ctx); u != nil {
		actorID = pgtype.Int8{Int64: u.ID, Valid: true}
		if u.Token != nil {
			tokenID = pgtype.Int8{Int64: u.Token.ID, Valid: true}
			tokenName = pgtype.Text{String: u.Token.Name, Valid: u.Token.Name != ""}
		}
	}

	q := dbgen.New(db)
	row, err := q.InsertActivityLog(ctx, dbgen.InsertActivityLogParams{
		SpaceSlug:  entry.SpaceSlug,
		ActorID:    actorID,
		TokenID:    tokenID,
		TokenName:  tokenName,
		EntityType: dbgen.ActivityEntityType(entry.EntityType),
		EntityID:   entry.EntityID,
		Action:     dbgen.ActivityAction(entry.Action),
		CreatedAt:  types.Timestamptz(now),
	})
	if err != nil {
		return err
	}

	for _, d := range entry.Details {
		var from, to pgtype.Text
		if d.From != nil {
			from = pgtype.Text{String: *d.From, Valid: true}
		}
		if d.To != nil {
			to = pgtype.Text{String: *d.To, Valid: true}
		}
		if err := q.InsertActivityLogDetail(ctx, dbgen.InsertActivityLogDetailParams{
			ActivityLogID: row.ID,
			Field:         d.Field,
			FromValue:     from,
			ToValue:       to,
		}); err != nil {
			return err
		}
	}

	return nil
}
