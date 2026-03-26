-- name: CreateAuthToken :one
INSERT INTO auth_tokens (user_id, token_hash, name, kind, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAuthTokenByHash :one
SELECT
    t.id,
    t.user_id,
    t.token_hash,
    t.name,
    t.kind,
    t.expires_at,
    t.created_at,
    u.id         AS user_id_2,
    u.email      AS user_email,
    u.name       AS user_name,
    u.is_owner   AS user_is_owner
FROM auth_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token_hash = $1;

-- name: ListAuthTokensByUser :many
SELECT * FROM auth_tokens
WHERE user_id = $1 AND id > $2
ORDER BY id ASC
LIMIT $3;

-- name: DeleteAuthToken :execresult
DELETE FROM auth_tokens WHERE id = $1 AND user_id = $2;

-- name: DeleteAuthTokenByHash :execresult
DELETE FROM auth_tokens WHERE token_hash = $1;
