package api

import (
	"context"
	"fmt"
	"sort"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// replaceLevelsOps defines the operations needed by replaceLevels for a
// particular level type.
type replaceLevelsOps[Item any] struct {
	label          string
	itemName       func(Item) string
	listNames      func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error)
	create         func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item Item, pos int64) error
	update         func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item Item, pos int64) error
	delete         func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error
	validateRemove func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error // nil = skip
}

// replaceLevels implements the shared diff-and-sync logic for status, effort,
// and priority level replacement.
func replaceLevels[Item any](ctx context.Context, q *dbgen.Queries, spaceSlug string, items []Item, ops replaceLevelsOps[Item]) error {
	// Check for duplicate names.
	names := make(map[string]struct{}, len(items))
	for _, item := range items {
		n := ops.itemName(item)
		if _, ok := names[n]; ok {
			return badRequest(fmt.Sprintf("duplicate %s name %q", ops.label, n))
		}
		names[n] = struct{}{}
	}

	// List current names.
	currentNamesList, err := ops.listNames(ctx, q, spaceSlug)
	if err != nil {
		return err
	}
	currentNames := make(map[string]struct{}, len(currentNamesList))
	for _, n := range currentNamesList {
		currentNames[n] = struct{}{}
	}

	// Compute removed names in stable order.
	var toRemove []string
	for _, n := range currentNamesList {
		if _, kept := names[n]; !kept {
			toRemove = append(toRemove, n)
		}
	}
	sort.Strings(toRemove)

	// Validate removals if a hook is provided.
	if ops.validateRemove != nil {
		for _, name := range toRemove {
			if err := ops.validateRemove(ctx, q, spaceSlug, name); err != nil {
				return err
			}
		}
	}

	// Delete removed levels.
	for _, name := range toRemove {
		if err := ops.delete(ctx, q, spaceSlug, name); err != nil {
			return err
		}
	}

	// Insert new and update existing.
	for i, item := range items {
		if _, exists := currentNames[ops.itemName(item)]; exists {
			if err := ops.update(ctx, q, spaceSlug, item, int64(i)); err != nil {
				return err
			}
		} else {
			if err := ops.create(ctx, q, spaceSlug, item, int64(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// --- Task Statuses ---

func (h *Handler) SpaceTaskStatusesList(ctx context.Context, params apigen.SpaceTaskStatusesListParams) (*apigen.TaskStatusList, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.DB)

	rows, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	items, err := convertEach(statusFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskStatusList{Items: items}, nil
}

func (h *Handler) SpaceTaskStatusesReplace(ctx context.Context, req *apigen.TaskStatusReplace, params apigen.SpaceTaskStatusesReplaceParams) (*apigen.TaskStatusList, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, apigen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	// Status-specific validation: at least 1 initial and 1 completion.
	var hasInitial, hasCompletion bool
	for _, item := range req.Items {
		switch item.Category {
		case apigen.TaskStatusInputCategoryInitial:
			hasInitial = true
		case apigen.TaskStatusInputCategoryIntermediate:
			// no constraint
		case apigen.TaskStatusInputCategoryCompletion:
			hasCompletion = true
		}
	}
	if !hasInitial {
		return nil, badRequest("at least one status with category \"initial\" is required")
	}
	if !hasCompletion {
		return nil, badRequest("at least one status with category \"completion\" is required")
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskStatusInput]{
		label:    "status",
		itemName: func(s apigen.TaskStatusInput) string { return s.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			names := make([]string, len(rows))
			for i, r := range rows {
				names[i] = r.Name
			}
			return names, nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskStatusInput, pos int64) error {
			_, err := q.CreateTaskStatus(ctx, dbgen.CreateTaskStatusParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Category:  string(item.Category),
				Position:  pos,
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskStatusInput, pos int64) error {
			return q.UpdateTaskStatus(ctx, dbgen.UpdateTaskStatusParams{
				Category:  string(item.Category),
				Position:  pos,
				SpaceSlug: spaceSlug,
				Name:      item.Name,
			})
		},
		delete: func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error {
			_, err := q.DeleteTaskStatus(ctx, dbgen.DeleteTaskStatusParams{
				SpaceSlug: spaceSlug,
				Name:      name,
			})
			return err
		},
		validateRemove: func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error {
			count, err := q.CountTasksByStatusName(ctx, dbgen.CountTasksByStatusNameParams{
				SpaceSlug:  spaceSlug,
				StatusName: name,
			})
			if err != nil {
				return err
			}
			if count > 0 {
				return badRequest(fmt.Sprintf("cannot remove status %q: %d task(s) still reference it", name, count))
			}
			return nil
		},
	}); err != nil {
		return nil, err
	}

	// Re-fetch and return.
	rows, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	items, err := convertEach(statusFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskStatusList{Items: items}, nil
}

// --- Task Effort Levels ---

func (h *Handler) SpaceTaskEffortLevelsList(ctx context.Context, params apigen.SpaceTaskEffortLevelsListParams) (*apigen.TaskEffortLevelList, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.DB)

	rows, err := q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	items, err := convertEach(effortLevelFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskEffortLevelList{Items: items}, nil
}

func (h *Handler) SpaceTaskEffortLevelsReplace(ctx context.Context, req *apigen.TaskEffortLevelReplace, params apigen.SpaceTaskEffortLevelsReplaceParams) (*apigen.TaskEffortLevelList, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, apigen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskEffortLevelInput]{
		label:    "effort level",
		itemName: func(e apigen.TaskEffortLevelInput) string { return e.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskEffortLevelsBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			names := make([]string, len(rows))
			for i, r := range rows {
				names[i] = r.Name
			}
			return names, nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskEffortLevelInput, pos int64) error {
			_, err := q.CreateTaskEffortLevel(ctx, dbgen.CreateTaskEffortLevelParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Position:  pos,
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskEffortLevelInput, pos int64) error {
			return q.UpdateTaskEffortLevel(ctx, dbgen.UpdateTaskEffortLevelParams{
				Position:  pos,
				SpaceSlug: spaceSlug,
				Name:      item.Name,
			})
		},
		delete: func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error {
			_, err := q.DeleteTaskEffortLevel(ctx, dbgen.DeleteTaskEffortLevelParams{
				SpaceSlug: spaceSlug,
				Name:      name,
			})
			return err
		},
	}); err != nil {
		return nil, err
	}

	// Re-fetch and return.
	rows, err := q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	items, err := convertEach(effortLevelFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskEffortLevelList{Items: items}, nil
}

// --- Task Priority Levels ---

func (h *Handler) SpaceTaskPriorityLevelsList(ctx context.Context, params apigen.SpaceTaskPriorityLevelsListParams) (*apigen.TaskPriorityLevelList, error) {
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.DB)

	rows, err := q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	items, err := convertEach(priorityLevelFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPriorityLevelList{Items: items}, nil
}

func (h *Handler) SpaceTaskPriorityLevelsReplace(ctx context.Context, req *apigen.TaskPriorityLevelReplace, params apigen.SpaceTaskPriorityLevelsReplaceParams) (*apigen.TaskPriorityLevelList, error) {
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, apigen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskPriorityLevelInput]{
		label:    "priority level",
		itemName: func(p apigen.TaskPriorityLevelInput) string { return p.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskPriorityLevelsBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			names := make([]string, len(rows))
			for i, r := range rows {
				names[i] = r.Name
			}
			return names, nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskPriorityLevelInput, pos int64) error {
			_, err := q.CreateTaskPriorityLevel(ctx, dbgen.CreateTaskPriorityLevelParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Position:  pos,
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskPriorityLevelInput, pos int64) error {
			return q.UpdateTaskPriorityLevel(ctx, dbgen.UpdateTaskPriorityLevelParams{
				Position:  pos,
				SpaceSlug: spaceSlug,
				Name:      item.Name,
			})
		},
		delete: func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error {
			_, err := q.DeleteTaskPriorityLevel(ctx, dbgen.DeleteTaskPriorityLevelParams{
				SpaceSlug: spaceSlug,
				Name:      name,
			})
			return err
		},
	}); err != nil {
		return nil, err
	}

	// Re-fetch and return.
	rows, err := q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	items, err := convertEach(priorityLevelFromDB)(rows)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPriorityLevelList{Items: items}, nil
}
