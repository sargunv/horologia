-- +goose Up

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_tasks_title_trgm
    ON tasks
    USING GIN (lower(title) gin_trgm_ops);

CREATE INDEX idx_space_members_user_space
    ON space_members (user_id, space_slug);

CREATE VIEW visible_tasks AS
SELECT
    sm.user_id AS viewer_user_id,
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.status_name,
    t.effort_name,
    t.priority_name,
    t.due_at,
    t.due_tz,
    t.recurrence_type,
    t.recurrence_rule,
    t.last_completed_at,
    t.created_at,
    t.updated_at,
    t.overdue_action_after_days,
    t.overdue_action,
    t.overdue_action_status
FROM tasks t
JOIN space_members sm ON sm.space_slug = t.space_slug

UNION

SELECT
    u.id AS viewer_user_id,
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.status_name,
    t.effort_name,
    t.priority_name,
    t.due_at,
    t.due_tz,
    t.recurrence_type,
    t.recurrence_rule,
    t.last_completed_at,
    t.created_at,
    t.updated_at,
    t.overdue_action_after_days,
    t.overdue_action,
    t.overdue_action_status
FROM tasks t
JOIN users u ON u.is_owner;

-- +goose Down

DROP VIEW IF EXISTS visible_tasks;
DROP INDEX IF EXISTS idx_space_members_user_space;
DROP INDEX IF EXISTS idx_tasks_title_trgm;
