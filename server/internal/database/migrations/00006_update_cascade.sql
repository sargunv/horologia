-- +goose NO TRANSACTION

-- +goose Up

-- Recreate tasks table to add ON UPDATE CASCADE on the status, effort,
-- and priority composite foreign keys. This allows level names to be
-- renamed in-place without breaking FK constraints.

PRAGMA foreign_keys = OFF;

-- Drop all triggers that reference the tasks table.
DROP TRIGGER IF EXISTS trg_reassign_tasks_on_status_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_effort_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_priority_delete;

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
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name) ON UPDATE CASCADE
);

INSERT INTO tasks_new (id, space_slug, title, description, status_name, effort_name, priority_name, due_date, created_at, updated_at)
SELECT id, space_slug, title, description, status_name, effort_name, priority_name, due_date, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_new RENAME TO tasks;

-- Recreate indexes.
CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);
CREATE UNIQUE INDEX idx_tasks_id_space ON tasks (id, space_slug);
CREATE INDEX idx_tasks_effort ON tasks (space_slug, effort_name)
    WHERE effort_name IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks (space_slug, priority_name)
    WHERE priority_name IS NOT NULL;

-- Recreate triggers.
-- Status deletion: prevent if tasks still reference the status.
-- +goose StatementBegin
CREATE TRIGGER trg_prevent_status_delete_if_tasks_exist
BEFORE DELETE ON task_statuses
FOR EACH ROW
WHEN (SELECT COUNT(*) FROM tasks WHERE space_slug = OLD.space_slug AND status_name = OLD.name) > 0
BEGIN
    SELECT RAISE(ABORT, 'cannot delete status: tasks still reference it');
END;
-- +goose StatementEnd

-- Effort deletion: nullify on tasks.
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

-- Priority deletion: nullify on tasks.
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

-- Advisory FK check (see 00005 for rationale).
PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;

-- +goose Down

PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_nullify_tasks_on_priority_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_effort_delete;
DROP TRIGGER IF EXISTS trg_prevent_status_delete_if_tasks_exist;

CREATE TABLE tasks_old (
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

INSERT INTO tasks_old (id, space_slug, title, description, status_name, effort_name, priority_name, due_date, created_at, updated_at)
SELECT id, space_slug, title, description, status_name, effort_name, priority_name, due_date, created_at, updated_at
FROM tasks;

DROP TABLE tasks;
ALTER TABLE tasks_old RENAME TO tasks;

CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);
CREATE UNIQUE INDEX idx_tasks_id_space ON tasks (id, space_slug);
CREATE INDEX idx_tasks_effort ON tasks (space_slug, effort_name)
    WHERE effort_name IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks (space_slug, priority_name)
    WHERE priority_name IS NOT NULL;

-- Restore original triggers from 00005.
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

PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;
