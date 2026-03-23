-- name: InsertTaskRelation :exec
INSERT INTO task_relations (source_task_id, target_task_id, space_slug, kind, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteTaskRelation :execresult
DELETE FROM task_relations
WHERE source_task_id = ? AND target_task_id = ? AND kind = ?;

-- name: ListRelationsByTaskAsSource :many
SELECT source_task_id, target_task_id, kind, created_at
FROM task_relations
WHERE source_task_id = ?
ORDER BY created_at ASC;

-- name: ListRelationsByTaskAsTarget :many
SELECT source_task_id, target_task_id, kind, created_at
FROM task_relations
WHERE target_task_id = ?
ORDER BY created_at ASC;
