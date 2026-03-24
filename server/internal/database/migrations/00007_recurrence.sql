-- +goose NO TRANSACTION

-- +goose Up

-- Add recurrence columns to tasks and 'triggers' kind to task_relations.
-- Requires table recreation for both:
-- - tasks: adding CHECK constraint on recurrence_type
-- - task_relations: widening the kind CHECK constraint

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

-- Recreate task_relations with 'triggers' added to the kind CHECK.
CREATE TABLE task_relations_new (
    source_task_id INTEGER NOT NULL,
    target_task_id INTEGER NOT NULL,
    space_slug     TEXT    NOT NULL,
    kind           TEXT    NOT NULL CHECK (kind IN ('parent', 'blocks', 'relates_to', 'duplicates', 'triggers')),
    created_at     INTEGER NOT NULL,
    PRIMARY KEY (source_task_id, target_task_id, kind),
    FOREIGN KEY (source_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE,
    FOREIGN KEY (target_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE,
    CHECK (source_task_id != target_task_id),
    CHECK (kind NOT IN ('relates_to', 'duplicates') OR source_task_id < target_task_id)
);

INSERT INTO task_relations_new (source_task_id, target_task_id, space_slug, kind, created_at)
SELECT source_task_id, target_task_id, space_slug, kind, created_at
FROM task_relations;

DROP TABLE task_relations;
ALTER TABLE task_relations_new RENAME TO task_relations;

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);

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
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name) ON UPDATE CASCADE
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

-- Restore task_relations without 'triggers' kind.
CREATE TABLE task_relations_old (
    source_task_id INTEGER NOT NULL,
    target_task_id INTEGER NOT NULL,
    space_slug     TEXT    NOT NULL,
    kind           TEXT    NOT NULL CHECK (kind IN ('parent', 'blocks', 'relates_to', 'duplicates')),
    created_at     INTEGER NOT NULL,
    PRIMARY KEY (source_task_id, target_task_id, kind),
    FOREIGN KEY (source_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE,
    FOREIGN KEY (target_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE,
    CHECK (source_task_id != target_task_id),
    CHECK (kind NOT IN ('relates_to', 'duplicates') OR source_task_id < target_task_id)
);

INSERT INTO task_relations_old (source_task_id, target_task_id, space_slug, kind, created_at)
SELECT source_task_id, target_task_id, space_slug, kind, created_at
FROM task_relations
WHERE kind != 'triggers';

DROP TABLE task_relations;
ALTER TABLE task_relations_old RENAME TO task_relations;

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);

PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;
