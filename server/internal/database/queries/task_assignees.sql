-- name: ListAssigneeUserIDsByTask :many
SELECT user_id FROM task_assignees
WHERE task_id = ?
ORDER BY user_id ASC;

-- name: ListAssigneeUserIDsByTasks :many
SELECT task_id, user_id FROM task_assignees
WHERE task_id IN (sqlc.slice('task_ids'))
ORDER BY task_id, user_id;

-- name: InsertTaskAssignee :exec
INSERT INTO task_assignees (task_id, user_id, created_at)
VALUES (?, ?, ?);

-- name: DeleteTaskAssignees :exec
DELETE FROM task_assignees WHERE task_id = ?;

-- name: DeleteTaskAssigneesBySpaceAndUser :exec
DELETE FROM task_assignees
WHERE user_id = ?
  AND task_id IN (SELECT id FROM tasks WHERE space_slug = ?);
