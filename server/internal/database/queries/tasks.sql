-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, due_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasksBySpace :many
SELECT * FROM tasks
WHERE space_slug = ? AND id > ?
ORDER BY id ASC
LIMIT ?;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, status_name = ?, due_date = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = ?;
