-- +goose Up

DROP TRIGGER trg_nullify_tasks_on_effort_delete ON task_effort_levels;
DROP TRIGGER trg_nullify_tasks_on_priority_delete ON task_priority_levels;
DROP FUNCTION fn_nullify_tasks_on_effort_delete();
DROP FUNCTION fn_nullify_tasks_on_priority_delete();

ALTER TABLE tasks
DROP CONSTRAINT tasks_space_slug_effort_name_fkey,
DROP CONSTRAINT tasks_space_slug_priority_name_fkey,
ADD CONSTRAINT tasks_space_slug_effort_name_fkey
    FOREIGN KEY (space_slug, effort_name)
    REFERENCES task_effort_levels (space_slug, name)
    ON UPDATE CASCADE
    ON DELETE SET NULL (effort_name),
ADD CONSTRAINT tasks_space_slug_priority_name_fkey
    FOREIGN KEY (space_slug, priority_name)
    REFERENCES task_priority_levels (space_slug, name)
    ON UPDATE CASCADE
    ON DELETE SET NULL (priority_name);

-- +goose Down

ALTER TABLE tasks
DROP CONSTRAINT tasks_space_slug_effort_name_fkey,
DROP CONSTRAINT tasks_space_slug_priority_name_fkey,
ADD CONSTRAINT tasks_space_slug_effort_name_fkey
    FOREIGN KEY (space_slug, effort_name)
    REFERENCES task_effort_levels (space_slug, name)
    ON UPDATE CASCADE,
ADD CONSTRAINT tasks_space_slug_priority_name_fkey
    FOREIGN KEY (space_slug, priority_name)
    REFERENCES task_priority_levels (space_slug, name)
    ON UPDATE CASCADE;

-- +goose StatementBegin
CREATE FUNCTION fn_nullify_tasks_on_effort_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE tasks
    SET effort_name = NULL, updated_at = NOW()
    WHERE space_slug = OLD.space_slug AND effort_name = OLD.name;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_nullify_tasks_on_effort_delete
BEFORE DELETE ON task_effort_levels
FOR EACH ROW
EXECUTE FUNCTION fn_nullify_tasks_on_effort_delete();

-- +goose StatementBegin
CREATE FUNCTION fn_nullify_tasks_on_priority_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE tasks
    SET priority_name = NULL, updated_at = NOW()
    WHERE space_slug = OLD.space_slug AND priority_name = OLD.name;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_nullify_tasks_on_priority_delete
BEFORE DELETE ON task_priority_levels
FOR EACH ROW
EXECUTE FUNCTION fn_nullify_tasks_on_priority_delete();
