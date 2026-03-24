package api

import (
	"context"
	"fmt"
	"sort"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

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
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}

	// Validate: no duplicate names, at least 1 initial and 1 completion.
	names := make(map[string]struct{}, len(req.Items))
	var hasInitial, hasCompletion bool
	for _, item := range req.Items {
		if _, ok := names[item.Name]; ok {
			return nil, badRequest(fmt.Sprintf("duplicate status name %q", item.Name))
		}
		names[item.Name] = struct{}{}
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

	current, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	// Compute removed names in stable order.
	currentNames := make(map[string]struct{}, len(current))
	for _, s := range current {
		currentNames[s.Name] = struct{}{}
	}

	var toRemove []string
	for name := range currentNames {
		if _, kept := names[name]; !kept {
			toRemove = append(toRemove, name)
		}
	}
	sort.Strings(toRemove)

	// Validate all removals before deleting any.
	for _, name := range toRemove {
		count, err := q.CountTasksByStatusName(ctx, dbgen.CountTasksByStatusNameParams{
			SpaceSlug:  params.SpaceSlug,
			StatusName: name,
		})
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, badRequest(fmt.Sprintf("cannot remove status %q: %d task(s) still reference it", name, count))
		}
	}

	// Delete removed statuses.
	for _, name := range toRemove {
		if _, err := q.DeleteTaskStatus(ctx, dbgen.DeleteTaskStatusParams{
			SpaceSlug: params.SpaceSlug,
			Name:      name,
		}); err != nil {
			return nil, err
		}
	}

	// Insert new and update existing.
	for i, item := range req.Items {
		if _, exists := currentNames[item.Name]; exists {
			if err := q.UpdateTaskStatus(ctx, dbgen.UpdateTaskStatusParams{
				Category:  string(item.Category),
				Position:  int64(i),
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
			}); err != nil {
				return nil, err
			}
		} else {
			if _, err := q.CreateTaskStatus(ctx, dbgen.CreateTaskStatusParams{
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
				Category:  string(item.Category),
				Position:  int64(i),
			}); err != nil {
				return nil, err
			}
		}
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
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}

	// Validate: no duplicate names.
	names := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		if _, ok := names[item.Name]; ok {
			return nil, badRequest(fmt.Sprintf("duplicate effort level name %q", item.Name))
		}
		names[item.Name] = struct{}{}
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	current, err := q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	currentNames := make(map[string]struct{}, len(current))
	for _, e := range current {
		currentNames[e.Name] = struct{}{}
	}

	// Delete removed levels (triggers null out task references).
	for _, e := range current {
		if _, kept := names[e.Name]; !kept {
			if _, err := q.DeleteTaskEffortLevel(ctx, dbgen.DeleteTaskEffortLevelParams{
				SpaceSlug: params.SpaceSlug,
				Name:      e.Name,
			}); err != nil {
				return nil, err
			}
		}
	}

	// Insert new and update existing.
	for i, item := range req.Items {
		if _, exists := currentNames[item.Name]; exists {
			if err := q.UpdateTaskEffortLevel(ctx, dbgen.UpdateTaskEffortLevelParams{
				Position:  int64(i),
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
			}); err != nil {
				return nil, err
			}
		} else {
			if _, err := q.CreateTaskEffortLevel(ctx, dbgen.CreateTaskEffortLevelParams{
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
				Position:  int64(i),
			}); err != nil {
				return nil, err
			}
		}
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
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, "admin"); err != nil {
		return nil, err
	}

	// Validate: no duplicate names.
	names := make(map[string]struct{}, len(req.Items))
	for _, item := range req.Items {
		if _, ok := names[item.Name]; ok {
			return nil, badRequest(fmt.Sprintf("duplicate priority level name %q", item.Name))
		}
		names[item.Name] = struct{}{}
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	q := dbgen.New(tx)

	current, err := q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	currentNames := make(map[string]struct{}, len(current))
	for _, p := range current {
		currentNames[p.Name] = struct{}{}
	}

	// Delete removed levels (triggers null out task references).
	for _, p := range current {
		if _, kept := names[p.Name]; !kept {
			if _, err := q.DeleteTaskPriorityLevel(ctx, dbgen.DeleteTaskPriorityLevelParams{
				SpaceSlug: params.SpaceSlug,
				Name:      p.Name,
			}); err != nil {
				return nil, err
			}
		}
	}

	// Insert new and update existing.
	for i, item := range req.Items {
		if _, exists := currentNames[item.Name]; exists {
			if err := q.UpdateTaskPriorityLevel(ctx, dbgen.UpdateTaskPriorityLevelParams{
				Position:  int64(i),
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
			}); err != nil {
				return nil, err
			}
		} else {
			if _, err := q.CreateTaskPriorityLevel(ctx, dbgen.CreateTaskPriorityLevelParams{
				SpaceSlug: params.SpaceSlug,
				Name:      item.Name,
				Position:  int64(i),
			}); err != nil {
				return nil, err
			}
		}
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
