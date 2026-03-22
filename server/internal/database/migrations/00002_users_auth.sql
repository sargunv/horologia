-- +goose Up

CREATE TABLE users (
    id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    email         TEXT    NOT NULL UNIQUE,
    name          TEXT    NOT NULL,
    password_hash TEXT,
    is_owner      INTEGER NOT NULL DEFAULT 0,
    oidc_subject  TEXT    UNIQUE,
    created_at    TEXT    NOT NULL,
    updated_at    TEXT    NOT NULL
);

CREATE TABLE auth_tokens (
    id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL DEFAULT '',
    kind       TEXT    NOT NULL CHECK (kind IN ('session', 'api')),
    expires_at TEXT,
    created_at TEXT    NOT NULL
);

CREATE INDEX idx_auth_tokens_user ON auth_tokens (user_id);

CREATE TABLE space_members (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       TEXT    NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
    created_at TEXT    NOT NULL,
    PRIMARY KEY (space_slug, user_id)
);

CREATE INDEX idx_space_members_user ON space_members (user_id);

-- +goose Down

DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS users;
