-- +goose NO TRANSACTION

-- +goose Up

CREATE TABLE task_effort_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (space_slug, name)
);

CREATE TABLE task_priority_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (space_slug, name)
);

-- Recreate tasks table to add composite FKs for effort and priority.
-- SQLite ALTER TABLE ADD COLUMN cannot add composite foreign keys.
-- Follow SQLite's documented table-alteration procedure: disable FKs,
-- recreate the table, check FK integrity, re-enable FKs.
--
-- WARNING: This migration runs outside a transaction (required for PRAGMA
-- foreign_keys changes). A crash mid-migration leaves the DB in a broken
-- state. Take a backup before running.

PRAGMA foreign_keys = OFF;

-- Drop trigger that references tasks table before recreating it.
DROP TRIGGER IF EXISTS trg_reassign_tasks_on_status_delete;

CREATE TABLE tasks_new (
    id            INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    space_slug    TEXT    NOT NULL,
    title         TEXT    NOT NULL,
    description   TEXT    NOT NULL DEFAULT '',
    status_name   TEXT    NOT NULL,
    effort_name   TEXT,
    priority_name TEXT,
    due_date      TEXT,
    created_at    INTEGER NOT NULL,
    updated_at    INTEGER NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name),
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name),
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name)
);

INSERT INTO tasks_new (id, space_slug, title, description, status_name, due_date, created_at, updated_at)
SELECT id, space_slug, title, description, status_name, due_date, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

-- Recreate indexes that were on the original tasks table.
CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);
CREATE UNIQUE INDEX idx_tasks_id_space ON tasks (id, space_slug);
CREATE INDEX idx_tasks_effort ON tasks (space_slug, effort_name)
    WHERE effort_name IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks (space_slug, priority_name)
    WHERE priority_name IS NOT NULL;

-- Recreate trigger for status reassignment on delete.
-- +goose StatementBegin
CREATE TRIGGER trg_reassign_tasks_on_status_delete
BEFORE DELETE ON task_statuses
FOR EACH ROW
BEGIN
    UPDATE tasks
    SET status_name = (
        SELECT name FROM task_statuses
        WHERE space_slug = OLD.space_slug AND category = 'initial'
        LIMIT 1
    ),
    updated_at = unixepoch()
    WHERE space_slug = OLD.space_slug AND status_name = OLD.name;
END;
-- +goose StatementEnd

-- Triggers to nullify effort/priority on level deletion.
-- Cannot use ON DELETE SET NULL on composite FKs because SQLite would try
-- to NULL all columns in the FK tuple, including the NOT NULL space_slug.
-- +goose StatementBegin
CREATE TRIGGER trg_nullify_tasks_on_effort_delete
BEFORE DELETE ON task_effort_levels
FOR EACH ROW
BEGIN
    UPDATE tasks
    SET effort_name = NULL, updated_at = unixepoch()
    WHERE space_slug = OLD.space_slug AND effort_name = OLD.name;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_nullify_tasks_on_priority_delete
BEFORE DELETE ON task_priority_levels
FOR EACH ROW
BEGIN
    UPDATE tasks
    SET priority_name = NULL, updated_at = unixepoch()
    WHERE space_slug = OLD.space_slug AND priority_name = OLD.name;
END;
-- +goose StatementEnd

-- Advisory: PRAGMA foreign_key_check returns violation rows but does not raise
-- an error. Goose discards the result set. This serves as documentation that
-- FK integrity was considered, not as an enforced gate.
PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;

-- +goose Down

PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_nullify_tasks_on_priority_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_effort_delete;
DROP TRIGGER IF EXISTS trg_reassign_tasks_on_status_delete;

CREATE TABLE tasks_old (
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

INSERT INTO tasks_old (id, space_slug, title, description, status_name, due_date, created_at, updated_at)
SELECT id, space_slug, title, description, status_name, due_date, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);
CREATE UNIQUE INDEX idx_tasks_id_space ON tasks (id, space_slug);

-- Recreate the trigger from migration 00003.
-- +goose StatementBegin
CREATE TRIGGER trg_reassign_tasks_on_status_delete
BEFORE DELETE ON task_statuses
FOR EACH ROW
BEGIN
    UPDATE tasks
    SET status_name = (
        SELECT name FROM task_statuses
        WHERE space_slug = OLD.space_slug AND category = 'initial'
        LIMIT 1
    ),
    updated_at = unixepoch()
    WHERE space_slug = OLD.space_slug AND status_name = OLD.name;
END;
-- +goose StatementEnd

-- Advisory: see comment in Up migration.
PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;

DROP TABLE IF EXISTS task_priority_levels;
DROP TABLE IF EXISTS task_effort_levels;
