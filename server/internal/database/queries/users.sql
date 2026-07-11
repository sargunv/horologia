-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, is_owner, oidc_subject, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByOIDCSubject :one
SELECT * FROM users WHERE oidc_subject = $1;

-- name: SetUserOIDCSubject :exec
UPDATE users SET oidc_subject = $1, updated_at = $2 WHERE id = $3;

-- name: ListUsers :many
SELECT * FROM users ORDER BY id ASC;

-- name: UpdateUser :one
UPDATE users
SET name = $1,
    email = $2,
    is_owner = $3,
    appearance_mode = $4,
    appearance_light_theme = $5,
    appearance_dark_theme = $6,
    updated_at = $7
WHERE id = $8
RETURNING *;

-- name: UpdateUserPasswordHash :exec
UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3;

-- name: DeleteUser :execresult
DELETE FROM users WHERE id = $1;

-- name: CountOwners :one
SELECT COUNT(*) FROM users WHERE is_owner = TRUE;
