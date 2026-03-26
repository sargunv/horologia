-- name: CreateSpace :one
INSERT INTO spaces (slug, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetSpace :one
SELECT * FROM spaces WHERE slug = $1;

-- name: ListSpaces :many
SELECT * FROM spaces
ORDER BY slug ASC;

-- name: UpdateSpace :one
UPDATE spaces
SET slug = $1, name = $2, description = $3, updated_at = $4
WHERE slug = $5
RETURNING *;

-- name: DeleteSpace :execresult
DELETE FROM spaces WHERE slug = $1;
