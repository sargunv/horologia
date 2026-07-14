-- +goose Up

CREATE VIEW visible_recipes AS
SELECT
    sm.user_id AS viewer_user_id,
    r.*
FROM recipes r
JOIN space_members sm ON sm.space_slug = r.space_slug

UNION

SELECT
    u.id AS viewer_user_id,
    r.*
FROM recipes r
JOIN users u ON u.is_owner;

DROP INDEX idx_activity_log_task;
CREATE INDEX idx_activity_log_entity
    ON activity_log (entity_type, entity_id, space_slug, id);

-- +goose Down

DROP INDEX idx_activity_log_entity;
CREATE INDEX idx_activity_log_task
    ON activity_log (entity_type, entity_id, space_slug, id)
    WHERE entity_type = 'task';
DROP VIEW visible_recipes;
