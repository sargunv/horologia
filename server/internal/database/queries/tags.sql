-- name: CreateTag :one
INSERT INTO tags (space_slug, name, name_folded, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: EnsureTag :one
INSERT INTO tags (space_slug, name, name_folded, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT (space_slug, name_folded) DO UPDATE SET name = name
RETURNING *;

-- name: GetTagByFoldedName :one
SELECT * FROM tags
WHERE space_slug = ? AND name_folded = ?;

-- name: ListTagsBySpace :many
SELECT * FROM tags
WHERE space_slug = ? AND id > ?
ORDER BY id ASC
LIMIT ?;

-- name: UpdateTag :one
UPDATE tags
SET name = ?, name_folded = ?
WHERE id = ? AND space_slug = ?
RETURNING *;

-- name: DeleteTag :execresult
DELETE FROM tags WHERE space_slug = ? AND name_folded = ?;
