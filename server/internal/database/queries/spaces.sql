-- name: CreateSpace :one
INSERT INTO spaces (slug, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSpace :one
SELECT * FROM spaces WHERE slug = ?;

-- name: ListSpaces :many
SELECT * FROM spaces
WHERE slug > ?
ORDER BY slug ASC
LIMIT ?;

-- name: UpdateSpace :one
UPDATE spaces
SET name = ?, description = ?, updated_at = ?
WHERE slug = ?
RETURNING *;

-- name: DeleteSpace :execresult
DELETE FROM spaces WHERE slug = ?;
