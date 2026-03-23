-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, is_owner, oidc_subject, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByOIDCSubject :one
SELECT * FROM users WHERE oidc_subject = ?;

-- name: SetUserOIDCSubject :exec
UPDATE users SET oidc_subject = ?, updated_at = ? WHERE id = ?;
