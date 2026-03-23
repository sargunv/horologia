-- +goose Up

CREATE INDEX idx_auth_tokens_expires ON auth_tokens (expires_at)
    WHERE expires_at IS NOT NULL;

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

-- +goose Down

DROP TRIGGER IF EXISTS trg_reassign_tasks_on_status_delete;
DROP INDEX IF EXISTS idx_auth_tokens_expires;
