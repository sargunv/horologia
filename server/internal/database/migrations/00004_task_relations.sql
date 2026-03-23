-- +goose Up

-- Required for composite FK references (source/target_task_id, space_slug) -> (id, space_slug).
-- SQLite FK enforcement requires a UNIQUE index on exactly the referenced column set.
-- tasks.id is already PK but SQLite does not infer uniqueness of (id, space_slug) from that alone.
CREATE UNIQUE INDEX idx_tasks_id_space ON tasks (id, space_slug);

CREATE TABLE task_relations (
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

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);

-- +goose Down

DROP TABLE IF EXISTS task_relations;
DROP INDEX IF EXISTS idx_tasks_id_space;
