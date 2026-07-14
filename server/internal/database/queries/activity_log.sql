-- name: InsertActivityLog :one
INSERT INTO activity_log (space_slug, actor_id, token_id, token_name, entity_type, entity_id, action, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: InsertActivityLogDetail :exec
INSERT INTO activity_log_details (activity_log_id, field, from_value, to_value)
VALUES ($1, $2, $3, $4);

-- name: ListActivityLogBySpace :many
-- Cursor sentinel: 0 means "no cursor" (first page). BIGSERIAL IDs start at 1,
-- so 0 is never a real ID. Other list queries use `id > $cursor` where `id > 0`
-- naturally returns all rows; here the DESC order requires an explicit guard.
SELECT * FROM activity_log
WHERE space_slug = $1 AND ($2::bigint = 0 OR id < $2)
ORDER BY id DESC
LIMIT $3;

-- name: ListActivityLogByTask :many
SELECT * FROM activity_log
WHERE entity_type = 'task' AND entity_id = $1 AND space_slug = $2 AND ($3::bigint = 0 OR id < $3)
ORDER BY id DESC
LIMIT $4;

-- name: ListActivityLogByRecipe :many
SELECT * FROM activity_log
WHERE entity_type = 'recipe' AND entity_id = $1 AND space_slug = $2 AND ($3::bigint = 0 OR id < $3)
ORDER BY id DESC
LIMIT $4;

-- name: ListActivityLogByActor :many
SELECT al.* FROM activity_log al
WHERE al.actor_id = $1
  AND ($2::bigint = 0 OR al.id < $2)
  AND (
    $4::boolean
    OR EXISTS (SELECT 1 FROM space_members sm WHERE sm.space_slug = al.space_slug AND sm.user_id = $3)
  )
ORDER BY al.id DESC
LIMIT $5;

-- name: ListActivityLogDetailsByLogIDs :many
SELECT * FROM activity_log_details
WHERE activity_log_id = ANY($1::bigint[])
ORDER BY activity_log_id, id;
