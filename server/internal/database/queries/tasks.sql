-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, effort_name, priority_name, due_at, due_tz, recurrence_type, recurrence_rule, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ? AND space_slug = ?;

-- name: ListTasksBySpace :many
SELECT * FROM tasks
WHERE space_slug = ? AND id > ?
ORDER BY id ASC
LIMIT ?;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, status_name = ?, effort_name = ?, priority_name = ?, due_at = ?, due_tz = ?,
    recurrence_type = ?, recurrence_rule = ?, last_completed_at = ?, updated_at = ?
WHERE id = ? AND space_slug = ?
RETURNING *;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = ? AND space_slug = ?;

-- name: ResetTaskToInitial :exec
UPDATE tasks
SET status_name = ?, updated_at = ?
WHERE id = ? AND space_slug = ? AND status_name != ?;

-- name: ConvertAccumulatingToOneOff :execresult
UPDATE tasks
SET recurrence_type = 'one_off', recurrence_rule = NULL, updated_at = ?
WHERE id = ? AND space_slug = ? AND recurrence_type = 'fixed_accumulating';

-- name: ListOverdueAccumulatingTasks :many
SELECT * FROM tasks
WHERE recurrence_type = 'fixed_accumulating'
  AND due_at IS NOT NULL
  AND due_at <= ?
ORDER BY space_slug, id ASC
LIMIT 100;
