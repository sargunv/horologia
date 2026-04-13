-- name: CreateTaskEffortLevel :one
INSERT INTO task_effort_levels (space_slug, name, position, icon)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTaskEffortLevelsBySpace :many
SELECT * FROM task_effort_levels
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskEffortLevel :exec
UPDATE task_effort_levels SET position = $1, icon = $2
WHERE space_slug = $3 AND name = $4;

-- name: DeleteTaskEffortLevel :execresult
DELETE FROM task_effort_levels
WHERE space_slug = $1 AND name = $2;
