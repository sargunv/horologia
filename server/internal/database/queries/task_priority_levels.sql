-- name: CreateTaskPriorityLevel :one
INSERT INTO task_priority_levels (space_slug, name, position)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListTaskPriorityLevelsBySpace :many
SELECT * FROM task_priority_levels
WHERE space_slug = ?
ORDER BY position ASC;
