-- name: CreateTaskStatus :one
INSERT INTO task_statuses (space_slug, name, category, position)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListTaskStatusesBySpace :many
SELECT * FROM task_statuses
WHERE space_slug = ?
ORDER BY position ASC;

-- name: UpdateTaskStatus :exec
UPDATE task_statuses SET category = ?, position = ?
WHERE space_slug = ? AND name = ?;

-- name: DeleteTaskStatus :execresult
DELETE FROM task_statuses
WHERE space_slug = ? AND name = ?;

-- name: CountTasksByStatusName :one
SELECT COUNT(*) FROM tasks
WHERE space_slug = ? AND status_name = ?;
