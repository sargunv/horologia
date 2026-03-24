-- +goose NO TRANSACTION

-- +goose Up

-- Add 'spawns' to the task_relations kind CHECK constraint.
-- Requires table recreation (SQLite limitation).

PRAGMA foreign_keys = OFF;

CREATE TABLE task_relations_new (
    source_task_id INTEGER NOT NULL,
    target_task_id INTEGER NOT NULL,
    space_slug     TEXT    NOT NULL,
    kind           TEXT    NOT NULL CHECK (kind IN ('parent', 'blocks', 'relates_to', 'duplicates', 'triggers', 'spawns')),
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

-- Partial index for the fixed_accumulating cron query.
CREATE INDEX idx_tasks_overdue_accumulating
    ON tasks (due_at)
    WHERE recurrence_type = 'fixed_accumulating' AND due_at IS NOT NULL;

PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;

-- +goose Down

PRAGMA foreign_keys = OFF;

DROP INDEX IF EXISTS idx_tasks_overdue_accumulating;

CREATE TABLE task_relations_old (
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

INSERT INTO task_relations_old (source_task_id, target_task_id, space_slug, kind, created_at)
SELECT source_task_id, target_task_id, space_slug, kind, created_at
FROM task_relations
WHERE kind != 'spawns';

DROP TABLE task_relations;
ALTER TABLE task_relations_old RENAME TO task_relations;

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);

PRAGMA foreign_key_check;

PRAGMA foreign_keys = ON;
