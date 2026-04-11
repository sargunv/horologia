-- name: GetOAuthClient :one
SELECT * FROM oauth_clients
WHERE client_id = $1;

-- name: CreateOAuthAuthorizationCode :one
INSERT INTO oauth_authorization_codes (
    code_hash,
    user_id,
    client_id,
    redirect_uri,
    scopes,
    resource,
    code_challenge,
    code_challenge_method,
    expires_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetOAuthAuthorizationCodeByHash :one
SELECT * FROM oauth_authorization_codes
WHERE code_hash = $1;

-- name: DeleteOAuthAuthorizationCodeByHash :execresult
DELETE FROM oauth_authorization_codes
WHERE code_hash = $1;

-- name: UpsertOAuthConsentGrant :one
INSERT INTO oauth_consent_grants (
    user_id,
    client_id,
    scope_key,
    scopes,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, client_id, scope_key) DO UPDATE SET
    scopes = EXCLUDED.scopes,
    updated_at = EXCLUDED.updated_at
RETURNING *;

-- name: GetOAuthConsentGrant :one
SELECT * FROM oauth_consent_grants
WHERE user_id = $1
  AND client_id = $2
  AND scope_key = $3;
