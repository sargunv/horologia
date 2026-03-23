-- name: CreateTaskStatus :one
INSERT INTO task_statuses (space_slug, name, category, position)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: ListTaskStatusesBySpace :many
SELECT * FROM task_statuses
WHERE space_slug = ?
ORDER BY position ASC;
