-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, effort_name, priority_name, due_at, due_tz, recurrence_type, recurrence_rule, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND space_slug = $2;

-- name: ListTasksBySpace :many
SELECT * FROM tasks
WHERE space_slug = $1 AND id > $2
ORDER BY id ASC
LIMIT $3;

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
SELECT * FROM tasks
WHERE recurrence_type = 'fixed_accumulating'
  AND due_at IS NOT NULL
  AND due_at <= $1
ORDER BY space_slug, id ASC
LIMIT 100;
