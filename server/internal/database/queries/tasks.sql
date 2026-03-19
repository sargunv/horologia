-- name: CreateTask :one
INSERT INTO tasks (space_slug, title, description, status_name, due_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTaskWithStatus :one
SELECT
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.due_date,
    t.created_at,
    t.updated_at,
    ts.name     AS status_name,
    ts.category AS status_category
FROM tasks t
JOIN task_statuses ts ON ts.space_slug = t.space_slug AND ts.name = t.status_name
WHERE t.id = ?;

-- name: ListTasksBySpace :many
SELECT
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.due_date,
    t.created_at,
    t.updated_at,
    ts.name     AS status_name,
    ts.category AS status_category
FROM tasks t
JOIN task_statuses ts ON ts.space_slug = t.space_slug AND ts.name = t.status_name
WHERE t.space_slug = ? AND t.id > ?
ORDER BY t.id ASC
LIMIT ?;

-- name: UpdateTask :one
UPDATE tasks
SET title = ?, description = ?, status_name = ?, due_date = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteTask :execresult
DELETE FROM tasks WHERE id = ?;
