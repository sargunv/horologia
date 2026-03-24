-- +goose Up

CREATE TABLE task_rotation_pool (
    task_id    INTEGER NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (task_id, user_id),
    UNIQUE (task_id, position)
);

CREATE INDEX idx_task_rotation_pool_user ON task_rotation_pool (user_id);

-- +goose Down

DROP TABLE IF EXISTS task_rotation_pool;
