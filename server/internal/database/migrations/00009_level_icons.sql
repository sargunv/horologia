-- +goose Up

ALTER TABLE task_effort_levels ADD COLUMN icon TEXT;
ALTER TABLE task_priority_levels ADD COLUMN icon TEXT;

-- +goose Down

ALTER TABLE task_effort_levels DROP COLUMN icon;
ALTER TABLE task_priority_levels DROP COLUMN icon;
