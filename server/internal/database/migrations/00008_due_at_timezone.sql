-- +goose NO TRANSACTION

-- +goose Up

-- Replace due_date (TEXT, YYYY-MM-DD) with due_at (INTEGER, epoch seconds)
-- and due_tz (TEXT, IANA timezone). Both must be null or both non-null.

PRAGMA foreign_keys = OFF;

-- Drop all triggers referencing tasks.
DROP TRIGGER IF EXISTS trg_prevent_status_delete_if_tasks_exist;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_effort_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_priority_delete;

CREATE TABLE tasks_new (
    id              INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    space_slug      TEXT    NOT NULL,
    title           TEXT    NOT NULL,
    description     TEXT    NOT NULL DEFAULT '',
    status_name     TEXT    NOT NULL,
    effort_name     TEXT,
    priority_name   TEXT,
    due_at          INTEGER,
    due_tz          TEXT,
    recurrence_type TEXT    NOT NULL DEFAULT 'one_off'
        CHECK (recurrence_type IN ('one_off', 'completion_based', 'fixed_non_accumulating', 'fixed_accumulating', 'on_dependency')),
    recurrence_rule TEXT,
    last_completed_at INTEGER,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name) ON UPDATE CASCADE,
    CHECK (
        (recurrence_type IN ('one_off', 'on_dependency') AND recurrence_rule IS NULL)
        OR
        (recurrence_type IN ('completion_based', 'fixed_non_accumulating', 'fixed_accumulating') AND recurrence_rule IS NOT NULL)
    ),
    CHECK ((due_at IS NULL AND due_tz IS NULL) OR (due_at IS NOT NULL AND due_tz IS NOT NULL))
);

-- Migrate existing due_date (YYYY-MM-DD) to due_at (epoch seconds at midnight UTC).
-- Existing data has no timezone info, so we use UTC as a reasonable default.
INSERT INTO tasks_new (
    id, space_slug, title, description, status_name, effort_name, priority_name,
    due_at, due_tz,
    recurrence_type, recurrence_rule, last_completed_at, created_at, updated_at
)
SELECT
    id, space_slug, title, description, status_name, effort_name, priority_name,
    CASE WHEN due_date IS NOT NULL THEN CAST(strftime('%s', due_date) AS INTEGER) ELSE NULL END,
    CASE WHEN due_date IS NOT NULL THEN 'UTC' ELSE NULL END,
    recurrence_type, recurrence_rule, last_completed_at, created_at, updated_at
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
-- +goose StatementBegin
CREATE TRIGGER trg_prevent_status_delete_if_tasks_exist
BEFORE DELETE ON task_statuses
FOR EACH ROW
WHEN (SELECT COUNT(*) FROM tasks WHERE space_slug = OLD.space_slug AND status_name = OLD.name) > 0
BEGIN
    SELECT RAISE(ABORT, 'cannot delete status: tasks still reference it');
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

-- +goose Down

PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_nullify_tasks_on_priority_delete;
DROP TRIGGER IF EXISTS trg_nullify_tasks_on_effort_delete;
DROP TRIGGER IF EXISTS trg_prevent_status_delete_if_tasks_exist;

CREATE TABLE tasks_old (
    id              INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    space_slug      TEXT    NOT NULL,
    title           TEXT    NOT NULL,
    description     TEXT    NOT NULL DEFAULT '',
    status_name     TEXT    NOT NULL,
    effort_name     TEXT,
    priority_name   TEXT,
    due_date        TEXT,
    recurrence_type TEXT    NOT NULL DEFAULT 'one_off'
        CHECK (recurrence_type IN ('one_off', 'completion_based', 'fixed_non_accumulating', 'fixed_accumulating', 'on_dependency')),
    recurrence_rule TEXT,
    last_completed_at INTEGER,
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name) ON UPDATE CASCADE,
    CHECK (
        (recurrence_type IN ('one_off', 'on_dependency') AND recurrence_rule IS NULL)
        OR
        (recurrence_type IN ('completion_based', 'fixed_non_accumulating', 'fixed_accumulating') AND recurrence_rule IS NOT NULL)
    )
);

-- Convert back: due_at epoch to due_date YYYY-MM-DD string (in UTC).
INSERT INTO tasks_old (
    id, space_slug, title, description, status_name, effort_name, priority_name,
    due_date,
    recurrence_type, recurrence_rule, last_completed_at, created_at, updated_at
)
SELECT
    id, space_slug, title, description, status_name, effort_name, priority_name,
    CASE WHEN due_at IS NOT NULL THEN date(due_at, 'unixepoch') ELSE NULL END,
    recurrence_type, recurrence_rule, last_completed_at, created_at, updated_at
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

-- +goose StatementBegin
CREATE TRIGGER trg_prevent_status_delete_if_tasks_exist
BEFORE DELETE ON task_statuses
FOR EACH ROW
WHEN (SELECT COUNT(*) FROM tasks WHERE space_slug = OLD.space_slug AND status_name = OLD.name) > 0
BEGIN
    SELECT RAISE(ABORT, 'cannot delete status: tasks still reference it');
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
