-- +goose Up

CREATE TYPE activity_entity_type AS ENUM (
    'task',
    'space',
    'member',
    'tag',
    'status',
    'effort_level',
    'priority_level',
    'relation'
);

CREATE TYPE activity_action AS ENUM ('created', 'updated', 'deleted');

CREATE TABLE activity_log (
    id          BIGSERIAL            NOT NULL PRIMARY KEY,
    space_slug  TEXT                 NOT NULL,
    actor_id    BIGINT,
    token_id    BIGINT,
    token_name  TEXT,
    entity_type activity_entity_type NOT NULL,
    entity_id   TEXT                 NOT NULL,
    action      activity_action      NOT NULL,
    created_at  TIMESTAMPTZ          NOT NULL
);

CREATE INDEX idx_activity_log_space ON activity_log (space_slug, id);
CREATE INDEX idx_activity_log_task  ON activity_log (entity_type, entity_id, space_slug, id)
    WHERE entity_type = 'task';
CREATE INDEX idx_activity_log_actor ON activity_log (actor_id, id)
    WHERE actor_id IS NOT NULL;

CREATE TABLE activity_log_details (
    id              BIGSERIAL NOT NULL PRIMARY KEY,
    activity_log_id BIGINT    NOT NULL REFERENCES activity_log (id),
    field           TEXT      NOT NULL,
    from_value      TEXT,
    to_value        TEXT
);

CREATE INDEX idx_activity_log_details_log ON activity_log_details (activity_log_id);

-- +goose Down

DROP TABLE IF EXISTS activity_log_details;
DROP TABLE IF EXISTS activity_log;
DROP TYPE IF EXISTS activity_action;
DROP TYPE IF EXISTS activity_entity_type;
