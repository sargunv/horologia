-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, effort_name, priority_name, due_date, recurrence_type, recurrence_rule, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
SET title = ?, description = ?, status_name = ?, effort_name = ?, priority_name = ?, due_date = ?,
    recurrence_type = ?, recurrence_rule = ?, last_completed_at = ?, updated_at = ?
WHERE id = ? AND space_slug = ?
RETURNING *;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = ? AND space_slug = ?;

-- name: ResetTaskToInitial :exec
UPDATE tasks
SET status_name = ?, updated_at = ?
WHERE id = ? AND space_slug = ? AND status_name != ?;
