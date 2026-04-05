-- +goose Up

CREATE TYPE overdue_action AS ENUM (
    'advance_recurrence',
    'set_status',
    'clear_due_date'
);

ALTER TABLE tasks
    ADD COLUMN overdue_action_after_days INTEGER,
    ADD COLUMN overdue_action             overdue_action,
    ADD COLUMN overdue_action_status      TEXT,
    ADD CONSTRAINT chk_overdue_action_rule CHECK (
        (overdue_action_after_days IS NULL) = (overdue_action IS NULL)
    ),
    ADD CONSTRAINT chk_overdue_action_status CHECK (
        (overdue_action = 'set_status') = (overdue_action_status IS NOT NULL)
    );

CREATE INDEX idx_tasks_overdue_action
    ON tasks (due_at, overdue_action_after_days)
    WHERE overdue_action IS NOT NULL
      AND due_at IS NOT NULL
      AND recurrence_type NOT IN ('one_off', 'on_dependency');

-- +goose Down

DROP INDEX IF EXISTS idx_tasks_overdue_action;

ALTER TABLE tasks
    DROP CONSTRAINT IF EXISTS chk_overdue_action_status,
    DROP CONSTRAINT IF EXISTS chk_overdue_action_rule,
    DROP COLUMN IF EXISTS overdue_action_status,
    DROP COLUMN IF EXISTS overdue_action,
    DROP COLUMN IF EXISTS overdue_action_after_days;

DROP TYPE IF EXISTS overdue_action;
