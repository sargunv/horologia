-- name: CreateTaskPriorityLevel :one
INSERT INTO task_priority_levels (space_slug, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTaskPriorityLevelsBySpace :many
SELECT * FROM task_priority_levels
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskPriorityLevel :exec
UPDATE task_priority_levels SET position = $1
WHERE space_slug = $2 AND name = $3;

-- name: DeleteTaskPriorityLevel :execresult
DELETE FROM task_priority_levels
WHERE space_slug = $1 AND name = $2;
