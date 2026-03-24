-- name: CreateTaskEffortLevel :one
INSERT INTO task_effort_levels (space_slug, name, position)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListTaskEffortLevelsBySpace :many
SELECT * FROM task_effort_levels
WHERE space_slug = ?
ORDER BY position ASC;
