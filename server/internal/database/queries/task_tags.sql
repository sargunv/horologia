-- name: ListTagNamesByTask :many
SELECT tg.name FROM task_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.task_id = ?
ORDER BY tg.name ASC;

-- name: InsertTaskTag :exec
INSERT INTO task_tags (task_id, tag_id, created_at)
VALUES (?, ?, ?);

-- name: DeleteTaskTags :exec
DELETE FROM task_tags WHERE task_id = ?;
