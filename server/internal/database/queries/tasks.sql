-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, effort_name, priority_name, due_at, due_tz, recurrence_type, recurrence_rule, overdue_action_after_days, overdue_action, overdue_action_status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND space_slug = $2;

-- name: GetTaskForUpdate :one
SELECT * FROM tasks WHERE id = $1 AND space_slug = $2 FOR UPDATE;

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
    recurrence_type = $8, recurrence_rule = $9, last_completed_at = $10,
    overdue_action_after_days = $11, overdue_action = $12, overdue_action_status = $13, updated_at = $14
WHERE id = $15 AND space_slug = $16
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

-- name: ListTasksWithOverdueActionDue :many
-- Returns recurring tasks whose overdue action grace period has elapsed.
-- Cap at 100 rows per tick to provide backpressure on the cron job.
SELECT * FROM tasks
WHERE overdue_action IS NOT NULL
  AND due_at IS NOT NULL
  AND recurrence_type NOT IN ('one_off', 'on_dependency')
  AND (due_at + COALESCE(overdue_action_after_days, 0)) <= $1
ORDER BY space_slug, id ASC
LIMIT 100;

-- name: SearchVisibleTasks :many
SELECT
    vt.id,
    vt.space_slug,
    vt.title,
    vt.status_name,
    vt.updated_at
FROM visible_tasks vt
WHERE
    vt.viewer_user_id = sqlc.arg(viewer_user_id)
    AND (sqlc.arg(space_slug) = '' OR vt.space_slug = sqlc.arg(space_slug))
    AND (sqlc.arg(exclude_task_id) = 0 OR vt.id != sqlc.arg(exclude_task_id))
    AND (
        (sqlc.arg(exact_task_id) != 0 AND vt.id = sqlc.arg(exact_task_id))
        OR (
            sqlc.arg(query_text) != ''
            AND (
                lower(vt.title) LIKE '%' || lower(sqlc.arg(query_text)) || '%'
                OR lower(vt.title) % lower(sqlc.arg(query_text))
            )
        )
    )
ORDER BY
    CASE
        WHEN sqlc.arg(exact_task_id) != 0 AND vt.id = sqlc.arg(exact_task_id) THEN 0
        WHEN lower(vt.title) = lower(sqlc.arg(query_text)) THEN 1
        WHEN lower(vt.title) LIKE lower(sqlc.arg(query_text)) || '%' THEN 2
        WHEN lower(vt.title) LIKE '%' || lower(sqlc.arg(query_text)) || '%' THEN 3
        ELSE 4
    END ASC,
    similarity(lower(vt.title), lower(sqlc.arg(query_text))) DESC,
    vt.updated_at DESC,
    vt.id DESC
LIMIT sqlc.arg(lim);

-- name: UpdateTaskOverdueActionAdvanceRecurrence :execresult
UPDATE tasks
SET due_at     = $1,
    due_tz     = $2,
    updated_at = $3
WHERE id = $4
  AND space_slug = $5
  AND overdue_action = 'advance_recurrence';

-- name: UpdateTaskOverdueActionSetStatus :execresult
UPDATE tasks
SET status_name = $1,
    updated_at  = $2
WHERE id = $3
  AND space_slug = $4
  AND overdue_action = 'set_status';

-- name: UpdateTaskOverdueActionClearDueDate :execresult
-- Clears the due date and removes the overdue action rule (since the rule
-- requires a due date, we must clear both together).
UPDATE tasks
SET due_at                    = NULL,
    due_tz                    = NULL,
    overdue_action_after_days = NULL,
    overdue_action            = NULL,
    overdue_action_status     = NULL,
    updated_at                = $1
WHERE id = $2
  AND space_slug = $3
  AND overdue_action = 'clear_due_date';

-- name: UpdateTaskOverdueActionExhausted :execresult
-- Called when advance_recurrence fires but ComputeNextDueAt returns nil
-- (recurrence rule exhausted). Clears due date and overdue action config.
UPDATE tasks
SET due_at                    = NULL,
    due_tz                    = NULL,
    overdue_action_after_days = NULL,
    overdue_action            = NULL,
    overdue_action_status     = NULL,
    updated_at                = $1
WHERE id = $2
  AND space_slug = $3
  AND overdue_action = 'advance_recurrence';
