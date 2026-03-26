-- name: ListTagNamesByTask :many
SELECT tg.name FROM task_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.task_id = $1
ORDER BY tg.name ASC;

-- name: ListTagNamesByTasks :many
SELECT tt.task_id, tg.name FROM task_tags tt
JOIN tags tg ON tg.id = tt.tag_id
WHERE tt.task_id = ANY(@task_ids::bigint[])
ORDER BY tt.task_id, tg.name;

-- name: InsertTaskTag :exec
INSERT INTO task_tags (task_id, tag_id, created_at)
VALUES ($1, $2, $3);

-- name: DeleteTaskTags :exec
DELETE FROM task_tags WHERE task_id = $1;
