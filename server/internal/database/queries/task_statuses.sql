-- name: CreateTaskStatus :one
INSERT INTO task_statuses (space_slug, name, category, position)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTaskStatusesBySpace :many
SELECT * FROM task_statuses
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskStatus :exec
UPDATE task_statuses SET category = $1, position = $2
WHERE space_slug = $3 AND name = $4;

-- name: DeleteTaskStatus :execresult
DELETE FROM task_statuses
WHERE space_slug = $1 AND name = $2;

-- name: CountTasksByStatusName :one
SELECT COUNT(*) FROM tasks
WHERE space_slug = $1 AND status_name = $2;
