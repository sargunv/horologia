-- name: CreateSpace :one
INSERT INTO spaces (slug, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSpace :one
SELECT * FROM spaces WHERE slug = $1;

-- name: ListSpaces :many
SELECT * FROM spaces
WHERE slug > $1
ORDER BY slug ASC
LIMIT $2;

-- name: UpdateSpace :one
UPDATE spaces
SET name = $1, description = $2, updated_at = $3
WHERE slug = $4
RETURNING *;

-- name: DeleteSpace :execresult
DELETE FROM spaces WHERE slug = $1;
