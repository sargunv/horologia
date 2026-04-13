-- name: CreateTaskPriorityLevel :one
INSERT INTO task_priority_levels (space_slug, name, position, icon)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTaskPriorityLevelsBySpace :many
SELECT * FROM task_priority_levels
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskPriorityLevel :exec
UPDATE task_priority_levels SET position = $1, icon = $2
WHERE space_slug = $3 AND name = $4;

-- name: DeleteTaskPriorityLevel :execresult
DELETE FROM task_priority_levels
WHERE space_slug = $1 AND name = $2;
