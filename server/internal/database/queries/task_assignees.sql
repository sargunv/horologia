-- name: ListAssigneeUserIDsByTask :many
SELECT user_id FROM task_assignees
WHERE task_id = $1
ORDER BY user_id ASC;

-- name: ListAssigneeUserIDsByTasks :many
SELECT task_id, user_id FROM task_assignees
WHERE task_id = ANY(@task_ids::bigint[])
ORDER BY task_id, user_id;

-- name: InsertTaskAssignee :exec
INSERT INTO task_assignees (task_id, user_id, created_at)
VALUES ($1, $2, $3);

-- name: DeleteTaskAssignees :exec
DELETE FROM task_assignees WHERE task_id = $1;

-- name: DeleteTaskAssigneesBySpaceAndUser :exec
DELETE FROM task_assignees
WHERE user_id = $1
  AND task_id IN (SELECT id FROM tasks WHERE space_slug = $2);
