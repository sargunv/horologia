-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, effort_name, priority_name, due_at, due_tz, recurrence_type, recurrence_rule, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND space_slug = $2;

-- name: ListTasksBySpace :many
SELECT
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
    (CASE ts.category
        WHEN 'completion' THEN ts.position
        ELSE -ts.position
    END)::integer AS sort_status,
    COALESCE(t.due_at, 'infinity'::date) AS sort_due,
    COALESCE(tpl.position, 2147483647) AS sort_priority,
    COALESCE(tel.position, 2147483647) AS sort_effort
FROM tasks t
JOIN task_statuses ts
    ON ts.space_slug = t.space_slug AND ts.name = t.status_name
LEFT JOIN task_priority_levels tpl
    ON tpl.space_slug = t.space_slug AND tpl.name = t.priority_name
LEFT JOIN task_effort_levels tel
    ON tel.space_slug = t.space_slug AND tel.name = t.effort_name
WHERE
    t.space_slug = @space_slug
    AND (
        (CASE ts.category WHEN 'completion' THEN ts.position ELSE -ts.position END)::integer,
        COALESCE(t.due_at, 'infinity'::date),
        COALESCE(tpl.position, 2147483647),
        COALESCE(tel.position, 2147483647),
        t.id
    ) > (
        @cursor_sort_status::integer,
        @cursor_sort_due::date,
        @cursor_sort_priority::integer,
        @cursor_sort_effort::integer,
        @cursor_id::bigint
    )
ORDER BY
    sort_status ASC,
    sort_due ASC,
    sort_priority ASC,
    sort_effort ASC,
    t.id ASC
LIMIT @lim;

-- name: ListTasksByUser :many
-- Lists all tasks assigned to a user across all spaces, with compound keyset
-- pagination. Sort order: status, due date, priority, effort, then task ID.
-- TODO: For cross-space comparability, normalize priority/effort positions to
-- 0.0–1.0 (position / max_position) instead of raw positions.
SELECT
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
    (CASE ts.category
        WHEN 'completion' THEN ts.position
        ELSE -ts.position
    END)::integer AS sort_status,
    COALESCE(t.due_at, 'infinity'::date) AS sort_due,
    COALESCE(tpl.position, 2147483647) AS sort_priority,
    COALESCE(tel.position, 2147483647) AS sort_effort
FROM tasks t
JOIN task_assignees ta
    ON ta.task_id = t.id
JOIN task_statuses ts
    ON ts.space_slug = t.space_slug AND ts.name = t.status_name
LEFT JOIN task_priority_levels tpl
    ON tpl.space_slug = t.space_slug AND tpl.name = t.priority_name
LEFT JOIN task_effort_levels tel
    ON tel.space_slug = t.space_slug AND tel.name = t.effort_name
WHERE
    ta.user_id = @assignee_user_id
    AND (
        (CASE ts.category WHEN 'completion' THEN ts.position ELSE -ts.position END)::integer,
        COALESCE(t.due_at, 'infinity'::date),
        COALESCE(tpl.position, 2147483647),
        COALESCE(tel.position, 2147483647),
        t.id
    ) > (
        @cursor_sort_status::integer,
        @cursor_sort_due::date,
        @cursor_sort_priority::integer,
        @cursor_sort_effort::integer,
        @cursor_id::bigint
    )
ORDER BY
    sort_status ASC,
    sort_due ASC,
    sort_priority ASC,
    sort_effort ASC,
    t.id ASC
LIMIT @lim;

-- name: UpdateTask :one
UPDATE tasks
SET title = $1, description = $2, status_name = $3, effort_name = $4, priority_name = $5, due_at = $6, due_tz = $7,
    recurrence_type = $8, recurrence_rule = $9, last_completed_at = $10, updated_at = $11
WHERE id = $12 AND space_slug = $13
RETURNING *;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = $1 AND space_slug = $2;

-- name: ResetTaskToInitial :exec
UPDATE tasks
SET status_name = $1, updated_at = $2
WHERE id = $3 AND space_slug = $4 AND status_name != $5;

-- name: ConvertAccumulatingToOneOff :execresult
UPDATE tasks
SET recurrence_type = 'one_off', recurrence_rule = NULL, updated_at = $1
WHERE id = $2 AND space_slug = $3 AND recurrence_type = 'fixed_accumulating';

-- name: ListOverdueAccumulatingTasks :many
-- Cap at 100 rows per tick to provide backpressure on the cron job.
-- The cron runs frequently enough that any remaining overdue tasks will
-- be picked up in subsequent ticks, so the backlog drains gradually.
SELECT * FROM tasks
WHERE recurrence_type = 'fixed_accumulating'
  AND due_at IS NOT NULL
  AND due_at <= $1
ORDER BY space_slug, id ASC
LIMIT 100;
