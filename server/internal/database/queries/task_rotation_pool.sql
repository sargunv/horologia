-- name: ListRotationPoolByTask :many
SELECT user_id FROM task_rotation_pool
WHERE task_id = $1
ORDER BY position ASC;

-- name: ListRotationPoolByTasks :many
SELECT task_id, user_id FROM task_rotation_pool
WHERE task_id = ANY(@task_ids::bigint[])
ORDER BY task_id, position ASC;

-- name: InsertRotationPoolMember :exec
INSERT INTO task_rotation_pool (task_id, user_id, position, created_at)
VALUES ($1, $2, $3, $4);

-- name: DeleteRotationPool :exec
DELETE FROM task_rotation_pool WHERE task_id = $1;

-- name: DeleteRotationPoolBySpaceAndUser :exec
DELETE FROM task_rotation_pool
WHERE user_id = $1
  AND task_id IN (SELECT id FROM tasks WHERE space_slug = $2);
