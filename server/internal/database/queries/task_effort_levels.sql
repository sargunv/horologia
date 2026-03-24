-- name: CreateTaskEffortLevel :one
INSERT INTO task_effort_levels (space_slug, name, position)
VALUES (?, ?, ?)
RETURNING *;

-- name: ListTaskEffortLevelsBySpace :many
SELECT * FROM task_effort_levels
WHERE space_slug = ?
ORDER BY position ASC;

-- name: UpdateTaskEffortLevel :exec
UPDATE task_effort_levels SET position = ?
WHERE space_slug = ? AND name = ?;

-- name: DeleteTaskEffortLevel :execresult
DELETE FROM task_effort_levels
WHERE space_slug = ? AND name = ?;
