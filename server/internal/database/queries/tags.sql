-- name: CreateTag :one
INSERT INTO tags (space_slug, name, name_folded, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: EnsureTag :one
-- Uses DO UPDATE SET name = tags.name (a true no-op that preserves the
-- existing display name) instead of DO NOTHING so that RETURNING returns
-- the row even when the insert is skipped due to conflict.
INSERT INTO tags (space_slug, name, name_folded, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (space_slug, name_folded) DO UPDATE SET name = tags.name
RETURNING *;

-- name: GetTagByFoldedName :one
SELECT * FROM tags
WHERE space_slug = $1 AND name_folded = $2;

-- name: ListAllTagsBySpace :many
SELECT * FROM tags
WHERE space_slug = $1
ORDER BY name ASC;

-- name: UpdateTag :one
UPDATE tags
SET name = $1, name_folded = $2
WHERE id = $3 AND space_slug = $4
RETURNING *;

-- name: DeleteTag :execresult
DELETE FROM tags WHERE space_slug = $1 AND name_folded = $2;
