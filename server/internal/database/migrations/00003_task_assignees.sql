-- +goose Up

CREATE TABLE task_assignees (
    task_id INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_assignees_user ON task_assignees (user_id);

-- +goose Down

DROP TABLE IF EXISTS task_assignees;
