-- +goose Up

ALTER TABLE task_effort_levels ADD COLUMN icon TEXT NOT NULL DEFAULT '';
ALTER TABLE task_priority_levels ADD COLUMN icon TEXT NOT NULL DEFAULT '';
ALTER TABLE task_statuses ADD COLUMN icon TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE task_effort_levels DROP COLUMN icon;
ALTER TABLE task_priority_levels DROP COLUMN icon;
ALTER TABLE task_statuses DROP COLUMN icon;
