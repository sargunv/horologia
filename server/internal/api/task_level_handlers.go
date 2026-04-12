package api

import (
	"context"
	"fmt"
	"time"

	apigen "github.com/sargunv/tend/api/gen"
	"github.com/sargunv/tend/server/internal/activitylog"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

// extractNames maps a slice of rows to a slice of name strings.
func extractNames[T any](rows []T, name func(T) string) []string {
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = name(r)
	}
	return names
}

// replaceLevelsOps defines the operations needed by replaceLevels for a
// particular level type.
type replaceLevelsOps[Item any] struct {
	label          string
	itemName       func(Item) string
	listNames      func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error)
	create         func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item Item, pos int32) error
	update         func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item Item, pos int32) error
	delete         func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error
	validateRemove func(ctx context.Context, q *dbgen.Queries, spaceSlug, name string) error // nil = skip
}

func validateStatusCategoryOrdering(items []apigen.TaskStatusInput) error {
	currentPhase := -1

	for _, item := range items {
		var phase int
		switch item.Category {
		case apigen.TaskStatusCategoryInitial:
			phase = 0
		case apigen.TaskStatusCategoryIntermediate:
			phase = 1
		case apigen.TaskStatusCategoryCompletion:
			phase = 2
		default:
			return badRequest(fmt.Sprintf("invalid status category %q", item.Category))
		}

		if phase < currentPhase {
			return badRequest(`status categories must be ordered "initial", then "intermediate", then "completion"`)
		}
		currentPhase = phase
	}

	return nil
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

	// Compute removed names in position order.
	var toRemove []string
	for _, n := range currentNamesList {
		if _, kept := names[n]; !kept {
			toRemove = append(toRemove, n)
		}
	}

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
			if err := ops.update(ctx, q, spaceSlug, item, int32(i)); err != nil {
				return err
			}
		} else {
			if err := ops.create(ctx, q, spaceSlug, item, int32(i)); err != nil {
				return err
			}
		}
	}

	return nil
}

// computeBulkDiff computes added and removed items between old and new name lists.
func computeBulkDiff(oldNames, newNames []string) (added, removed []string) {
	oldSet := make(map[string]struct{}, len(oldNames))
	for _, n := range oldNames {
		oldSet[n] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newNames))
	for _, n := range newNames {
		newSet[n] = struct{}{}
	}
	for _, n := range newNames {
		if _, ok := oldSet[n]; !ok {
			added = append(added, n)
		}
	}
	for _, n := range oldNames {
		if _, ok := newSet[n]; !ok {
			removed = append(removed, n)
		}
	}
	return added, removed
}

// --- Task Statuses ---

func (h *Handler) SpaceTaskStatusesList(ctx context.Context, params apigen.SpaceTaskStatusesListParams) (*apigen.TaskStatusList, error) {
	if err := h.requireScope(ctx, "spaces:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)

	rows, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskStatusList{Items: convertAll(rows, statusFromDB)}, nil
}

func (h *Handler) SpaceTaskStatusesReplace(ctx context.Context, req *apigen.TaskStatusReplace, params apigen.SpaceTaskStatusesReplaceParams) (*apigen.TaskStatusList, error) {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	// Status-specific validation: exactly 1 initial and at least 1 completion.
	var initialCount int
	var hasCompletion bool
	for _, item := range req.Items {
		switch item.Category {
		case apigen.TaskStatusCategoryInitial:
			initialCount++
		case apigen.TaskStatusCategoryIntermediate:
			// no constraint
		case apigen.TaskStatusCategoryCompletion:
			hasCompletion = true
		}
	}
	if initialCount != 1 {
		return nil, badRequest("exactly one status with category \"initial\" is required")
	}
	if !hasCompletion {
		return nil, badRequest("at least one status with category \"completion\" is required")
	}
	if err := validateStatusCategoryOrdering(req.Items); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Snapshot before for diff.
	beforeRows, err := q.ListTaskStatusesBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	beforeNames := extractNames(beforeRows, func(r dbgen.TaskStatus) string { return r.Name })

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskStatusInput]{
		label:    "status",
		itemName: func(s apigen.TaskStatusInput) string { return s.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskStatusesBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			return extractNames(rows, func(r dbgen.TaskStatus) string { return r.Name }), nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskStatusInput, pos int32) error {
			_, err := q.CreateTaskStatus(ctx, dbgen.CreateTaskStatusParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Category:  dbgen.StatusCategory(item.Category),
				Position:  pos,
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskStatusInput, pos int32) error {
			return q.UpdateTaskStatus(ctx, dbgen.UpdateTaskStatusParams{
				Category:  dbgen.StatusCategory(item.Category),
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

	afterNames := extractNames(rows, func(r dbgen.TaskStatus) string { return r.Name })
	added, removed := computeBulkDiff(beforeNames, afterNames)
	now := time.Now()
	for _, name := range added {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityStatus,
			EntityID:   name,
			Action:     activitylog.ActionCreated,
		}, now); err != nil {
			return nil, err
		}
	}
	for _, name := range removed {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityStatus,
			EntityID:   name,
			Action:     activitylog.ActionDeleted,
		}, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &apigen.TaskStatusList{Items: convertAll(rows, statusFromDB)}, nil
}

// --- Task Effort Levels ---

func (h *Handler) SpaceTaskEffortLevelsList(ctx context.Context, params apigen.SpaceTaskEffortLevelsListParams) (*apigen.TaskEffortLevelList, error) {
	if err := h.requireScope(ctx, "spaces:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)

	rows, err := q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskEffortLevelList{Items: convertAll(rows, effortLevelFromDB)}, nil
}

func (h *Handler) SpaceTaskEffortLevelsReplace(ctx context.Context, req *apigen.TaskEffortLevelReplace, params apigen.SpaceTaskEffortLevelsReplaceParams) (*apigen.TaskEffortLevelList, error) {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Validate icon fields (ogen does not enforce maxLength on nullable anyOf fields).
	for _, item := range req.Items {
		if err := validateIconField(item.Icon); err != nil {
			return nil, err
		}
	}

	// Snapshot before for diff.
	beforeRows, err := q.ListTaskEffortLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	beforeNames := extractNames(beforeRows, func(r dbgen.TaskEffortLevel) string { return r.Name })

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskEffortLevelInput]{
		label:    "effort level",
		itemName: func(e apigen.TaskEffortLevelInput) string { return e.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskEffortLevelsBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			return extractNames(rows, func(r dbgen.TaskEffortLevel) string { return r.Name }), nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskEffortLevelInput, pos int32) error {
			_, err := q.CreateTaskEffortLevel(ctx, dbgen.CreateTaskEffortLevelParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Position:  pos,
				Icon:      optNilStringToDBZero(item.Icon),
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskEffortLevelInput, pos int32) error {
			return q.UpdateTaskEffortLevel(ctx, dbgen.UpdateTaskEffortLevelParams{
				Position:  pos,
				Icon:      optNilStringToDBZero(item.Icon),
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

	afterNames := extractNames(rows, func(r dbgen.TaskEffortLevel) string { return r.Name })
	added, removed := computeBulkDiff(beforeNames, afterNames)
	now := time.Now()
	for _, name := range added {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityEffortLevel,
			EntityID:   name,
			Action:     activitylog.ActionCreated,
		}, now); err != nil {
			return nil, err
		}
	}
	for _, name := range removed {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityEffortLevel,
			EntityID:   name,
			Action:     activitylog.ActionDeleted,
		}, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &apigen.TaskEffortLevelList{Items: convertAll(rows, effortLevelFromDB)}, nil
}

// --- Task Priority Levels ---

func (h *Handler) SpaceTaskPriorityLevelsList(ctx context.Context, params apigen.SpaceTaskPriorityLevelsListParams) (*apigen.TaskPriorityLevelList, error) {
	if err := h.requireScope(ctx, "spaces:read"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRead(ctx, params.SpaceSlug); err != nil {
		return nil, err
	}

	q := dbgen.New(h.Pool)

	rows, err := q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}

	return &apigen.TaskPriorityLevelList{Items: convertAll(rows, priorityLevelFromDB)}, nil
}

func (h *Handler) SpaceTaskPriorityLevelsReplace(ctx context.Context, req *apigen.TaskPriorityLevelReplace, params apigen.SpaceTaskPriorityLevelsReplaceParams) (*apigen.TaskPriorityLevelList, error) {
	if err := h.requireScope(ctx, "spaces:write"); err != nil {
		return nil, err
	}
	if err := h.requireSpaceRole(ctx, params.SpaceSlug, dbgen.SpaceRoleAdmin); err != nil {
		return nil, err
	}

	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := dbgen.New(tx)

	// Validate icon fields (ogen does not enforce maxLength on nullable anyOf fields).
	for _, item := range req.Items {
		if err := validateIconField(item.Icon); err != nil {
			return nil, err
		}
	}

	// Snapshot before for diff.
	beforeRows, err := q.ListTaskPriorityLevelsBySpace(ctx, params.SpaceSlug)
	if err != nil {
		return nil, err
	}
	beforeNames := extractNames(beforeRows, func(r dbgen.TaskPriorityLevel) string { return r.Name })

	if err := replaceLevels(ctx, q, params.SpaceSlug, req.Items, replaceLevelsOps[apigen.TaskPriorityLevelInput]{
		label:    "priority level",
		itemName: func(p apigen.TaskPriorityLevelInput) string { return p.Name },
		listNames: func(ctx context.Context, q *dbgen.Queries, spaceSlug string) ([]string, error) {
			rows, err := q.ListTaskPriorityLevelsBySpace(ctx, spaceSlug)
			if err != nil {
				return nil, err
			}
			return extractNames(rows, func(r dbgen.TaskPriorityLevel) string { return r.Name }), nil
		},
		create: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskPriorityLevelInput, pos int32) error {
			_, err := q.CreateTaskPriorityLevel(ctx, dbgen.CreateTaskPriorityLevelParams{
				SpaceSlug: spaceSlug,
				Name:      item.Name,
				Position:  pos,
				Icon:      optNilStringToDBZero(item.Icon),
			})
			return err
		},
		update: func(ctx context.Context, q *dbgen.Queries, spaceSlug string, item apigen.TaskPriorityLevelInput, pos int32) error {
			return q.UpdateTaskPriorityLevel(ctx, dbgen.UpdateTaskPriorityLevelParams{
				Position:  pos,
				Icon:      optNilStringToDBZero(item.Icon),
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

	afterNames := extractNames(rows, func(r dbgen.TaskPriorityLevel) string { return r.Name })
	added, removed := computeBulkDiff(beforeNames, afterNames)
	now := time.Now()
	for _, name := range added {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityPriorityLevel,
			EntityID:   name,
			Action:     activitylog.ActionCreated,
		}, now); err != nil {
			return nil, err
		}
	}
	for _, name := range removed {
		if err := activitylog.Log(ctx, tx, activitylog.Entry{
			SpaceSlug:  params.SpaceSlug,
			EntityType: activitylog.EntityPriorityLevel,
			EntityID:   name,
			Action:     activitylog.ActionDeleted,
		}, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &apigen.TaskPriorityLevelList{Items: convertAll(rows, priorityLevelFromDB)}, nil
}
