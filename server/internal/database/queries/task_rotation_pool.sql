-- name: ListRotationPoolByTask :many
SELECT user_id FROM task_rotation_pool
WHERE task_id = ?
ORDER BY position ASC;

-- name: ListRotationPoolByTasks :many
SELECT task_id, user_id FROM task_rotation_pool
WHERE task_id IN (sqlc.slice('task_ids'))
ORDER BY task_id, position ASC;

-- name: InsertRotationPoolMember :exec
INSERT INTO task_rotation_pool (task_id, user_id, position, created_at)
VALUES (?, ?, ?, ?);

-- name: DeleteRotationPool :exec
DELETE FROM task_rotation_pool WHERE task_id = ?;

-- name: DeleteRotationPoolBySpaceAndUser :exec
DELETE FROM task_rotation_pool
WHERE user_id = ?
  AND task_id IN (SELECT id FROM tasks WHERE space_slug = ?);
