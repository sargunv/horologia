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
    -- Both null or both set (after_days acts as the presence sentinel alongside action).
    -- 0 = immediate (no grace period); positive = days of grace before action fires.
    ADD CONSTRAINT chk_overdue_action_rule CHECK (
        (overdue_action_after_days IS NULL) = (overdue_action IS NULL)
    ),
    ADD CONSTRAINT chk_overdue_action_after_days CHECK (
        overdue_action_after_days IS NULL OR overdue_action_after_days >= 0
    ),
    -- status column is present if and only if action is set_status.
    ADD CONSTRAINT chk_overdue_action_status CHECK (
        (overdue_action = 'set_status') = (overdue_action_status IS NOT NULL)
    ),
    -- Rule is only meaningful on recurring task types.
    ADD CONSTRAINT chk_overdue_action_recurring CHECK (
        overdue_action IS NULL
        OR recurrence_type NOT IN ('one_off'::recurrence_type, 'on_dependency'::recurrence_type)
    ),
    -- Rule requires a due date (the grace period is measured from due_at).
    ADD CONSTRAINT chk_overdue_action_requires_due CHECK (
        overdue_action IS NULL OR due_at IS NOT NULL
    );

-- Functional index: expression matches the query predicate
-- `(due_at + COALESCE(overdue_action_after_days, 0)) <= $today`.
CREATE INDEX idx_tasks_overdue_action
    ON tasks ((due_at + COALESCE(overdue_action_after_days, 0)))
    WHERE overdue_action IS NOT NULL
      AND due_at IS NOT NULL
      AND recurrence_type NOT IN ('one_off', 'on_dependency');

-- +goose Down

DROP INDEX IF EXISTS idx_tasks_overdue_action;

ALTER TABLE tasks
    DROP CONSTRAINT IF EXISTS chk_overdue_action_requires_due,
    DROP CONSTRAINT IF EXISTS chk_overdue_action_recurring,
    DROP CONSTRAINT IF EXISTS chk_overdue_action_status,
    DROP CONSTRAINT IF EXISTS chk_overdue_action_after_days,
    DROP CONSTRAINT IF EXISTS chk_overdue_action_rule,
    DROP COLUMN IF EXISTS overdue_action_status,
    DROP COLUMN IF EXISTS overdue_action,
    DROP COLUMN IF EXISTS overdue_action_after_days;

DROP TYPE IF EXISTS overdue_action;
