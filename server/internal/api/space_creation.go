package api

import (
	"context"
	"database/sql"
	"fmt"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
	"github.com/sargunv/tend/server/internal/types"
)

var defaultStatuses = []dbgen.CreateTaskStatusParams{
	{Name: "todo", Category: string(apigen.TaskStatusCategoryInitial), Position: 0},
	{Name: "done", Category: string(apigen.TaskStatusCategoryCompletion), Position: 1},
}

var defaultEffortLevels = []dbgen.CreateTaskEffortLevelParams{
	{Name: "small", Position: 0},
	{Name: "medium", Position: 1},
	{Name: "large", Position: 2},
}

var defaultPriorityLevels = []dbgen.CreateTaskPriorityLevelParams{
	{Name: "low", Position: 0},
	{Name: "medium", Position: 1},
	{Name: "high", Position: 2},
}

// CreateSpaceWithDefaults creates a space, its default statuses, effort levels,
// and priority levels, and adds the creator as an admin member, all in a single transaction.
func CreateSpaceWithDefaults(ctx context.Context, db *sql.DB, slug, name, description string, creatorUserID int64) (dbgen.Space, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return dbgen.Space{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)
	now := types.Now()

	space, err := q.CreateSpace(ctx, dbgen.CreateSpaceParams{
		Slug:        slug,
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
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
		Role:      string(apigen.SpaceRoleAdmin),
		CreatedAt: now,
	}); err != nil {
		return dbgen.Space{}, fmt.Errorf("create admin member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return dbgen.Space{}, fmt.Errorf("commit: %w", err)
	}

	return space, nil
}
