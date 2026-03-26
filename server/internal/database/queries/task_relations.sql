-- name: InsertTaskRelation :exec
INSERT INTO task_relations (source_task_id, target_task_id, space_slug, kind, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteTaskRelation :execresult
DELETE FROM task_relations
WHERE source_task_id = $1 AND target_task_id = $2 AND kind = $3 AND space_slug = $4;

-- name: ListRelationsByTasks :many
SELECT source_task_id, target_task_id, kind, created_at
FROM task_relations
WHERE space_slug = $1
  AND (source_task_id = ANY(@source_task_ids::bigint[])
    OR target_task_id = ANY(@target_task_ids::bigint[]))
ORDER BY created_at ASC;

-- Two single-task queries instead of one with OR, so each can use its own index
-- (PK for source, idx_task_relations_target for target). Used by fetchTask for
-- single-task reads; the batch ListRelationsByTasks is used for list endpoints.

-- name: ListRelationsByTaskAsSource :many
SELECT source_task_id, target_task_id, kind, created_at
FROM task_relations
WHERE source_task_id = $1 AND space_slug = $2
ORDER BY created_at ASC;

-- name: ListRelationsByTaskAsTarget :many
SELECT source_task_id, target_task_id, kind, created_at
FROM task_relations
WHERE target_task_id = $1 AND space_slug = $2
ORDER BY created_at ASC;

-- name: ListTriggerTargets :many
SELECT target_task_id
FROM task_relations
WHERE source_task_id = $1 AND space_slug = $2 AND kind = 'triggers';
