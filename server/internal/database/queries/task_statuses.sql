-- name: CreateTaskStatus :one
INSERT INTO task_statuses (space_slug, name, category, position, icon)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListTaskStatusesBySpace :many
SELECT * FROM task_statuses
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskStatus :exec
UPDATE task_statuses SET category = $1, position = $2, icon = $3
WHERE space_slug = $4 AND name = $5;

-- name: DeleteTaskStatus :execresult
DELETE FROM task_statuses
WHERE space_slug = $1 AND name = $2;

-- name: CountTasksByStatusName :one
SELECT COUNT(*) FROM tasks
WHERE space_slug = $1 AND status_name = $2;
