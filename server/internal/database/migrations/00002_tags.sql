-- +goose Up

CREATE TABLE tags (
    id          INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    space_slug  TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE,
    name        TEXT    NOT NULL,
    name_folded TEXT    NOT NULL,
    created_at  INTEGER NOT NULL,
    UNIQUE (space_slug, name_folded)
);

CREATE INDEX idx_tags_space_id ON tags (space_slug, id);

CREATE TABLE task_tags (
    task_id    INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    tag_id     INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

-- +goose Down

DROP TABLE IF EXISTS task_tags;
DROP TABLE IF EXISTS tags;
