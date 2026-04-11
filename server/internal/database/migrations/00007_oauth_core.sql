-- +goose Up
CREATE TABLE oauth_clients (
    client_id          TEXT        NOT NULL PRIMARY KEY,
    display_name       TEXT        NOT NULL,
    redirect_uris      TEXT[]      NOT NULL DEFAULT '{}',
    loopback_redirects BOOLEAN     NOT NULL DEFAULT FALSE,
    client_secret_hash TEXT,
    is_first_party     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth_authorization_codes (
    code_hash             TEXT        NOT NULL PRIMARY KEY,
    user_id               BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id             TEXT        NOT NULL REFERENCES oauth_clients (client_id) ON DELETE CASCADE,
    redirect_uri          TEXT        NOT NULL,
    scopes                TEXT[]      NOT NULL,
    resource              TEXT,
    code_challenge        TEXT        NOT NULL,
    code_challenge_method TEXT        NOT NULL,
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_authorization_codes_expires_at
    ON oauth_authorization_codes (expires_at);

CREATE TABLE oauth_consent_grants (
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id   TEXT        NOT NULL REFERENCES oauth_clients (client_id) ON DELETE CASCADE,
    scope_key   TEXT        NOT NULL,
    scopes      TEXT[]      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, client_id, scope_key)
);

INSERT INTO oauth_clients (
    client_id,
    display_name,
    redirect_uris,
    loopback_redirects,
    client_secret_hash,
    is_first_party,
    created_at
)
VALUES (
    'tend-cli',
    'Tend CLI',
    '{}',
    TRUE,
    NULL,
    TRUE,
    now()
)
ON CONFLICT (client_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS oauth_consent_grants;
DROP INDEX IF EXISTS idx_oauth_authorization_codes_expires_at;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_clients;
