-- +goose Up

CREATE TABLE spaces (
    slug        TEXT    NOT NULL PRIMARY KEY,
    name        TEXT    NOT NULL,
    description TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
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
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name)
);

CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);

-- +goose Down

DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS task_statuses;
DROP TABLE IF EXISTS spaces;
