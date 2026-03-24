package api

import (
	"context"

	apigen "github.com/sargunv/tend/server/api/gen"
	dbgen "github.com/sargunv/tend/server/internal/database/gen"
)

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
