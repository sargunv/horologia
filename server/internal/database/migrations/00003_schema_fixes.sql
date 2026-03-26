-- +goose Up

-- D3: Add space_slug to idx_task_relations_target so ListRelationsByTaskAsTarget
-- can use the index for both WHERE columns (target_task_id AND space_slug).
DROP INDEX idx_task_relations_target;
CREATE INDEX idx_task_relations_target ON task_relations (target_task_id, space_slug);

-- D4: Remove unreachable space_slug from idx_activity_log_actor. The id column
-- has a range predicate (id < $2), so B-tree traversal stops there; space_slug
-- at position 3 is never reached.
DROP INDEX idx_activity_log_actor;
CREATE INDEX idx_activity_log_actor ON activity_log (actor_id, id)
    WHERE actor_id IS NOT NULL;

-- D5: Bump tasks.updated_at when a status/effort/priority name is renamed via
-- ON UPDATE CASCADE. Without this, clients polling by updated_at miss the change.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_bump_tasks_updated_at_on_level_rename()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.name IS DISTINCT FROM NEW.name THEN
        EXECUTE format(
            'UPDATE tasks SET updated_at = NOW() WHERE space_slug = $1 AND %I = $2',
            TG_ARGV[0]
        ) USING NEW.space_slug, NEW.name;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_bump_tasks_updated_at_on_status_rename
AFTER UPDATE ON task_statuses
FOR EACH ROW
EXECUTE FUNCTION fn_bump_tasks_updated_at_on_level_rename('status_name');

CREATE TRIGGER trg_bump_tasks_updated_at_on_effort_rename
AFTER UPDATE ON task_effort_levels
FOR EACH ROW
EXECUTE FUNCTION fn_bump_tasks_updated_at_on_level_rename('effort_name');

CREATE TRIGGER trg_bump_tasks_updated_at_on_priority_rename
AFTER UPDATE ON task_priority_levels
FOR EACH ROW
EXECUTE FUNCTION fn_bump_tasks_updated_at_on_level_rename('priority_name');

-- +goose Down

DROP TRIGGER IF EXISTS trg_bump_tasks_updated_at_on_priority_rename ON task_priority_levels;
DROP TRIGGER IF EXISTS trg_bump_tasks_updated_at_on_effort_rename ON task_effort_levels;
DROP TRIGGER IF EXISTS trg_bump_tasks_updated_at_on_status_rename ON task_statuses;
DROP FUNCTION IF EXISTS fn_bump_tasks_updated_at_on_level_rename;

-- Restore original idx_activity_log_actor with space_slug
DROP INDEX IF EXISTS idx_activity_log_actor;
CREATE INDEX idx_activity_log_actor ON activity_log (actor_id, id, space_slug)
    WHERE actor_id IS NOT NULL;

-- Restore original idx_task_relations_target without space_slug
DROP INDEX IF EXISTS idx_task_relations_target;
CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);
