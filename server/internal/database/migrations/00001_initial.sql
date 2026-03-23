-- +goose Up

CREATE TABLE spaces (
    slug        TEXT    NOT NULL PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE TABLE task_statuses (
    space_slug TEXT NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    category   TEXT NOT NULL CHECK (category IN ('initial', 'intermediate', 'completion')),
    position   INTEGER NOT NULL,
    PRIMARY KEY (space_slug, name)
);

CREATE TABLE tasks (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    space_slug  TEXT    NOT NULL,
    title       TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    status_name TEXT    NOT NULL,
    due_date    TEXT,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name)
);

CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);

CREATE TABLE users (
    id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    email         TEXT    NOT NULL UNIQUE,
    name          TEXT    NOT NULL,
    password_hash TEXT,
    is_owner      INTEGER NOT NULL DEFAULT 0,
    oidc_subject  TEXT    UNIQUE,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL
);

CREATE TABLE auth_tokens (
    id         INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL DEFAULT '',
    kind       TEXT    NOT NULL CHECK (kind IN ('session', 'api')),
    expires_at INTEGER,
    created_at INTEGER NOT NULL
);

CREATE INDEX idx_auth_tokens_user ON auth_tokens (user_id);

CREATE TABLE space_members (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       TEXT    NOT NULL CHECK (role IN ('admin', 'member', 'viewer')),
    created_at INTEGER NOT NULL,
    PRIMARY KEY (space_slug, user_id)
);

CREATE INDEX idx_space_members_user ON space_members (user_id);

CREATE TABLE task_assignees (
    task_id    INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_assignees_user ON task_assignees (user_id);

-- +goose Down

DROP TABLE IF EXISTS task_assignees;
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS task_statuses;
DROP TABLE IF EXISTS spaces;
