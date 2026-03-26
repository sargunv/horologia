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
