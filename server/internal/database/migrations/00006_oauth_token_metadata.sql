-- +goose Up
ALTER TYPE auth_token_kind ADD VALUE IF NOT EXISTS 'oauth_access';
ALTER TYPE auth_token_kind ADD VALUE IF NOT EXISTS 'oauth_refresh';

ALTER TABLE auth_tokens
    ADD COLUMN oauth_client_id TEXT,
    ADD COLUMN oauth_scopes TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN oauth_resource TEXT;

ALTER TABLE auth_tokens
    ADD CONSTRAINT auth_tokens_delegated_metadata_chk CHECK (
        CASE
            WHEN kind IN ('session', 'api') THEN oauth_client_id IS NULL
                AND coalesce(array_length(oauth_scopes, 1), 0) = 0
                AND oauth_resource IS NULL
            ELSE oauth_client_id IS NOT NULL
        END
    );

CREATE INDEX idx_auth_tokens_oauth_client
    ON auth_tokens (oauth_client_id)
    WHERE oauth_client_id IS NOT NULL;

CREATE INDEX idx_auth_tokens_oauth_resource
    ON auth_tokens (oauth_resource)
    WHERE oauth_resource IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_auth_tokens_oauth_resource;
DROP INDEX IF EXISTS idx_auth_tokens_oauth_client;

ALTER TABLE auth_tokens
    DROP CONSTRAINT IF EXISTS auth_tokens_delegated_metadata_chk;

DELETE FROM auth_tokens
WHERE kind IN ('oauth_access', 'oauth_refresh');

ALTER TABLE auth_tokens
    DROP COLUMN IF EXISTS oauth_resource,
    DROP COLUMN IF EXISTS oauth_scopes,
    DROP COLUMN IF EXISTS oauth_client_id;

ALTER TABLE auth_tokens
    ALTER COLUMN kind TYPE TEXT;

DROP TYPE auth_token_kind;

CREATE TYPE auth_token_kind AS ENUM ('session', 'api');

ALTER TABLE auth_tokens
    ALTER COLUMN kind TYPE auth_token_kind USING kind::auth_token_kind;
