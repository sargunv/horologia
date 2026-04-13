package taskengine

import (
	"context"
	"fmt"
	"time"

	"github.com/sargunv/tend/server/internal/activitylog"
	"github.com/sargunv/tend/server/internal/database"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

var defaultStatuses = []dbgen.CreateTaskStatusParams{
	{Name: "todo", Category: dbgen.StatusCategoryInitial, Position: 0, Icon: "circle"},
	{Name: "done", Category: dbgen.StatusCategoryCompletion, Position: 1, Icon: "circle-check"},
}

var defaultEffortLevels = []dbgen.CreateTaskEffortLevelParams{
	{Name: "small", Position: 0, Icon: "feather"},
	{Name: "moderate", Position: 1, Icon: "gauge"},
	{Name: "large", Position: 2, Icon: "mountain"},
}

var defaultPriorityLevels = []dbgen.CreateTaskPriorityLevelParams{
	{Name: "low", Position: 0, Icon: "signal-low"},
	{Name: "medium", Position: 1, Icon: "signal-medium"},
	{Name: "high", Position: 2, Icon: "signal-high"},
}

// CreateSpaceWithDefaults creates a space, its default statuses, effort levels,
// and priority levels, and adds the creator as an admin member, all in a single transaction.
func CreateSpaceWithDefaults(ctx context.Context, db database.DB, slug, name, description string, creatorUserID int64, now time.Time) (dbgen.Space, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return dbgen.Space{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)
	nowTz := types.Timestamptz(now)

	space, err := q.CreateSpace(ctx, dbgen.CreateSpaceParams{
		Slug:        slug,
		Name:        name,
		Description: description,
		CreatedAt:   nowTz,
		UpdatedAt:   nowTz,
	})
	if err != nil {
		return dbgen.Space{}, fmt.Errorf("create space: %w", err)
	}

	for _, s := range defaultStatuses {
		s.SpaceSlug = space.Slug
		if _, err := q.CreateTaskStatus(ctx, s); err != nil {
			return dbgen.Space{}, fmt.Errorf("create default status %q: %w", s.Name, err)
		}
	}

	for _, e := range defaultEffortLevels {
		e.SpaceSlug = space.Slug
		if _, err := q.CreateTaskEffortLevel(ctx, e); err != nil {
			return dbgen.Space{}, fmt.Errorf("create default effort level %q: %w", e.Name, err)
		}
	}

	for _, p := range defaultPriorityLevels {
		p.SpaceSlug = space.Slug
		if _, err := q.CreateTaskPriorityLevel(ctx, p); err != nil {
			return dbgen.Space{}, fmt.Errorf("create default priority level %q: %w", p.Name, err)
		}
	}

	// Add creator as admin.
	if _, err := q.CreateSpaceMember(ctx, dbgen.CreateSpaceMemberParams{
		SpaceSlug: space.Slug,
		UserID:    creatorUserID,
		Role:      dbgen.SpaceRoleAdmin,
		CreatedAt: nowTz,
	}); err != nil {
		return dbgen.Space{}, fmt.Errorf("create admin member: %w", err)
	}

	// Log space creation inside the transaction for atomicity.
	if err := activitylog.Log(ctx, tx, activitylog.Entry{
		SpaceSlug:  space.Slug,
		EntityType: activitylog.EntitySpace,
		EntityID:   space.Slug,
		Action:     activitylog.ActionCreated,
		Details: []activitylog.Detail{
			{Field: "name", To: new(space.Name)},
		},
	}, now); err != nil {
		return dbgen.Space{}, fmt.Errorf("log space creation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return dbgen.Space{}, fmt.Errorf("commit: %w", err)
	}

	return space, nil
}
