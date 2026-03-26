-- name: CreateTaskEffortLevel :one
INSERT INTO task_effort_levels (space_slug, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTaskEffortLevelsBySpace :many
SELECT * FROM task_effort_levels
WHERE space_slug = $1
ORDER BY position ASC;

-- name: UpdateTaskEffortLevel :exec
UPDATE task_effort_levels SET position = $1
WHERE space_slug = $2 AND name = $3;

-- name: DeleteTaskEffortLevel :execresult
DELETE FROM task_effort_levels
WHERE space_slug = $1 AND name = $2;
