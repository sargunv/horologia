-- +goose Up

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Enum types
CREATE TYPE status_category AS ENUM ('initial', 'intermediate', 'completion');
CREATE TYPE space_role AS ENUM ('admin', 'member', 'viewer');
CREATE TYPE auth_token_kind AS ENUM ('session', 'api', 'oauth_access', 'oauth_refresh');
CREATE TYPE recurrence_type AS ENUM ('one_off', 'completion_based', 'fixed_non_accumulating', 'fixed_accumulating', 'on_dependency');
CREATE TYPE stored_relation_kind AS ENUM ('parent', 'blocks', 'relates_to', 'duplicates', 'triggers', 'spawns');
CREATE TYPE activity_entity_type AS ENUM (
    'task',
    'space',
    'member',
    'tag',
    'status',
    'effort_level',
    'priority_level',
    'relation',
    'user'
);
CREATE TYPE activity_action AS ENUM ('created', 'updated', 'deleted');
CREATE TYPE overdue_action AS ENUM (
    'advance_recurrence',
    'set_status',
    'clear_due_date'
);

-- Spaces
CREATE TABLE spaces (
    slug        TEXT        NOT NULL PRIMARY KEY,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);

-- Task statuses
CREATE TABLE task_statuses (
    space_slug TEXT            NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT            NOT NULL,
    category   status_category NOT NULL,
    position   INTEGER         NOT NULL,
    icon       TEXT            NOT NULL DEFAULT '',
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Task effort levels
CREATE TABLE task_effort_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    icon       TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Task priority levels
CREATE TABLE task_priority_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    icon       TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Tasks
CREATE TABLE tasks (
    id                        BIGSERIAL       NOT NULL PRIMARY KEY,
    space_slug                TEXT            NOT NULL,
    title                     TEXT            NOT NULL,
    description               TEXT            NOT NULL DEFAULT '',
    status_name               TEXT            NOT NULL,
    effort_name               TEXT,
    priority_name             TEXT,
    due_at                    DATE,
    due_tz                    TEXT,
    recurrence_type           recurrence_type NOT NULL DEFAULT 'one_off',
    recurrence_rule           TEXT,
    last_completed_at         TIMESTAMPTZ,
    created_at                TIMESTAMPTZ     NOT NULL,
    updated_at                TIMESTAMPTZ     NOT NULL,
    overdue_action_after_days INTEGER,
    overdue_action            overdue_action,
    overdue_action_status     TEXT,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, status_name) REFERENCES task_statuses (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, effort_name) REFERENCES task_effort_levels (space_slug, name) ON UPDATE CASCADE,
    FOREIGN KEY (space_slug, priority_name) REFERENCES task_priority_levels (space_slug, name) ON UPDATE CASCADE,
    CHECK (
        (recurrence_type IN ('one_off', 'on_dependency') AND recurrence_rule IS NULL)
        OR
        (recurrence_type IN ('completion_based', 'fixed_non_accumulating', 'fixed_accumulating') AND recurrence_rule IS NOT NULL)
    ),
    CHECK ((due_at IS NULL AND due_tz IS NULL) OR (due_at IS NOT NULL AND due_tz IS NOT NULL)),
    CONSTRAINT chk_overdue_action_rule CHECK (
        (overdue_action_after_days IS NULL) = (overdue_action IS NULL)
    ),
    CONSTRAINT chk_overdue_action_after_days CHECK (
        overdue_action_after_days IS NULL OR overdue_action_after_days >= 0
    ),
    CONSTRAINT chk_overdue_action_status CHECK (
        (overdue_action = 'set_status') = (overdue_action_status IS NOT NULL)
    ),
    CONSTRAINT chk_overdue_action_recurring CHECK (
        overdue_action IS NULL
        OR recurrence_type NOT IN ('one_off'::recurrence_type, 'on_dependency'::recurrence_type)
    ),
    CONSTRAINT chk_overdue_action_requires_due CHECK (
        overdue_action IS NULL OR due_at IS NOT NULL
    ),
    UNIQUE (id, space_slug)
);

CREATE INDEX idx_tasks_space ON tasks (space_slug);
CREATE INDEX idx_tasks_status ON tasks (space_slug, status_name);
CREATE INDEX idx_tasks_effort ON tasks (space_slug, effort_name)
    WHERE effort_name IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks (space_slug, priority_name)
    WHERE priority_name IS NOT NULL;
CREATE INDEX idx_tasks_overdue_accumulating
    ON tasks (due_at)
    WHERE recurrence_type = 'fixed_accumulating' AND due_at IS NOT NULL;
CREATE INDEX idx_tasks_overdue_action
    ON tasks ((due_at + COALESCE(overdue_action_after_days, 0)))
    WHERE overdue_action IS NOT NULL
      AND due_at IS NOT NULL
      AND recurrence_type NOT IN ('one_off', 'on_dependency');
CREATE INDEX idx_tasks_title_trgm
    ON tasks
    USING GIN (lower(title) gin_trgm_ops);

-- Users
CREATE TABLE users (
    id            BIGSERIAL   NOT NULL PRIMARY KEY,
    email         TEXT        NOT NULL UNIQUE,
    name          TEXT        NOT NULL,
    password_hash TEXT,
    is_owner      BOOLEAN     NOT NULL DEFAULT FALSE,
    oidc_subject  TEXT        UNIQUE,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

-- Auth tokens
CREATE TABLE auth_tokens (
    id              BIGSERIAL       NOT NULL PRIMARY KEY,
    user_id         BIGINT          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      TEXT            NOT NULL UNIQUE,
    name            TEXT            NOT NULL DEFAULT '',
    kind            auth_token_kind NOT NULL,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL,
    oauth_client_id TEXT,
    oauth_scopes    TEXT[]          NOT NULL DEFAULT '{}',
    oauth_resource  TEXT,
    CONSTRAINT auth_tokens_delegated_metadata_chk CHECK (
        CASE
            WHEN kind IN ('session', 'api') THEN oauth_client_id IS NULL
                AND coalesce(array_length(oauth_scopes, 1), 0) = 0
                AND oauth_resource IS NULL
            ELSE oauth_client_id IS NOT NULL
        END
    )
);

CREATE INDEX idx_auth_tokens_user ON auth_tokens (user_id);
CREATE INDEX idx_auth_tokens_expires ON auth_tokens (expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX idx_auth_tokens_oauth_client
    ON auth_tokens (oauth_client_id)
    WHERE oauth_client_id IS NOT NULL;
CREATE INDEX idx_auth_tokens_oauth_resource
    ON auth_tokens (oauth_resource)
    WHERE oauth_resource IS NOT NULL;

-- Space members
CREATE TABLE space_members (
    space_slug TEXT        NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       space_role  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (space_slug, user_id)
);

CREATE INDEX idx_space_members_user ON space_members (user_id);
CREATE INDEX idx_space_members_user_space ON space_members (user_id, space_slug);

-- Task assignees
CREATE TABLE task_assignees (
    task_id    BIGINT      NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_assignees_user ON task_assignees (user_id);

-- Tags
CREATE TABLE tags (
    id          BIGSERIAL   NOT NULL PRIMARY KEY,
    space_slug  TEXT        NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name        TEXT        NOT NULL,
    name_folded TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (space_slug, name_folded)
);

CREATE INDEX idx_tags_space_id ON tags (space_slug, id);

-- Task tags
CREATE TABLE task_tags (
    task_id    BIGINT      NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    tag_id     BIGINT      NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, tag_id)
);

CREATE INDEX idx_task_tags_tag ON task_tags (tag_id);

-- Task relations
CREATE TABLE task_relations (
    source_task_id BIGINT               NOT NULL,
    target_task_id BIGINT               NOT NULL,
    space_slug     TEXT                 NOT NULL,
    kind           stored_relation_kind NOT NULL,
    created_at     TIMESTAMPTZ          NOT NULL,
    PRIMARY KEY (source_task_id, target_task_id, kind),
    FOREIGN KEY (source_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (target_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE ON UPDATE CASCADE,
    CHECK (source_task_id != target_task_id),
    CHECK (kind NOT IN ('relates_to', 'duplicates') OR source_task_id < target_task_id)
);

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id, space_slug);
CREATE INDEX idx_task_relations_triggers ON task_relations (source_task_id, space_slug)
    WHERE kind = 'triggers';

-- Task rotation pool
CREATE TABLE task_rotation_pool (
    task_id    BIGINT      NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    position   INTEGER     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, user_id),
    UNIQUE (task_id, position)
);

CREATE INDEX idx_task_rotation_pool_user ON task_rotation_pool (user_id);

-- Activity log
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
CREATE INDEX idx_activity_log_task ON activity_log (entity_type, entity_id, space_slug, id)
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

-- OAuth
CREATE TABLE oauth_clients (
    client_id          TEXT        NOT NULL PRIMARY KEY,
    display_name       TEXT        NOT NULL,
    redirect_uris      TEXT[]      NOT NULL DEFAULT '{}',
    loopback_redirects BOOLEAN     NOT NULL DEFAULT FALSE,
    client_secret_hash TEXT,
    is_first_party     BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL
);

CREATE TABLE oauth_authorization_codes (
    code_hash             TEXT        NOT NULL PRIMARY KEY,
    user_id               BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id             TEXT        NOT NULL REFERENCES oauth_clients (client_id) ON DELETE CASCADE,
    redirect_uri          TEXT        NOT NULL,
    scopes                TEXT[]      NOT NULL,
    resource              TEXT,
    code_challenge        TEXT        NOT NULL,
    code_challenge_method TEXT        NOT NULL,
    expires_at            TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_oauth_authorization_codes_expires_at
    ON oauth_authorization_codes (expires_at);

CREATE TABLE oauth_consent_grants (
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    client_id   TEXT        NOT NULL REFERENCES oauth_clients (client_id) ON DELETE CASCADE,
    scope_key   TEXT        NOT NULL,
    scopes      TEXT[]      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, client_id, scope_key)
);

INSERT INTO oauth_clients (
    client_id,
    display_name,
    redirect_uris,
    loopback_redirects,
    client_secret_hash,
    is_first_party,
    created_at
)
VALUES (
    'horologia-cli',
    'Horologia CLI',
    '{}',
    TRUE,
    NULL,
    TRUE,
    now()
)
ON CONFLICT (client_id) DO NOTHING;

INSERT INTO oauth_clients (
    client_id,
    display_name,
    redirect_uris,
    loopback_redirects,
    client_secret_hash,
    is_first_party,
    created_at
)
VALUES (
    'horologia-mobile',
    'Horologia',
    '{horologia://oauth}',
    TRUE,
    NULL,
    TRUE,
    now()
)
ON CONFLICT (client_id) DO NOTHING;

-- Visible tasks view
CREATE VIEW visible_tasks AS
SELECT
    sm.user_id AS viewer_user_id,
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.status_name,
    t.effort_name,
    t.priority_name,
    t.due_at,
    t.due_tz,
    t.recurrence_type,
    t.recurrence_rule,
    t.last_completed_at,
    t.created_at,
    t.updated_at,
    t.overdue_action_after_days,
    t.overdue_action,
    t.overdue_action_status
FROM tasks t
JOIN space_members sm ON sm.space_slug = t.space_slug

UNION

SELECT
    u.id AS viewer_user_id,
    t.id,
    t.space_slug,
    t.title,
    t.description,
    t.status_name,
    t.effort_name,
    t.priority_name,
    t.due_at,
    t.due_tz,
    t.recurrence_type,
    t.recurrence_rule,
    t.last_completed_at,
    t.created_at,
    t.updated_at,
    t.overdue_action_after_days,
    t.overdue_action,
    t.overdue_action_status
FROM tasks t
JOIN users u ON u.is_owner;

-- Triggers

-- Prevent deleting a status that tasks still reference.
-- Skip the check when the owning space is being deleted (cascade).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_prevent_status_delete_if_tasks_exist()
RETURNS TRIGGER AS $$
BEGIN
    -- If this delete was triggered by a cascade (e.g. space deletion), allow it.
    -- pg_trigger_depth() > 1 means we're inside a nested trigger: the FK cascade
    -- internal trigger (depth 1) invoked this BEFORE DELETE trigger (depth 2).
    IF pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    IF EXISTS (SELECT 1 FROM tasks WHERE space_slug = OLD.space_slug AND status_name = OLD.name) THEN
        RAISE EXCEPTION 'cannot delete status: tasks still reference it';
    END IF;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_prevent_status_delete_if_tasks_exist
BEFORE DELETE ON task_statuses
FOR EACH ROW
EXECUTE FUNCTION fn_prevent_status_delete_if_tasks_exist();

-- Nullify effort on tasks when an effort level is deleted.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_nullify_tasks_on_effort_delete()
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

-- Nullify priority on tasks when a priority level is deleted.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION fn_nullify_tasks_on_priority_delete()
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

-- Bump tasks.updated_at when a status/effort/priority name is renamed via
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

DROP VIEW IF EXISTS visible_tasks;

DROP TABLE IF EXISTS oauth_consent_grants;
DROP INDEX IF EXISTS idx_oauth_authorization_codes_expires_at;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_clients;

DROP TABLE IF EXISTS activity_log_details;
DROP TABLE IF EXISTS activity_log;

DROP TABLE IF EXISTS task_rotation_pool;
DROP TABLE IF EXISTS task_relations;
DROP TABLE IF EXISTS task_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS task_assignees;
DROP TABLE IF EXISTS space_members;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS task_priority_levels;
DROP TABLE IF EXISTS task_effort_levels;
DROP TABLE IF EXISTS task_statuses;
DROP TABLE IF EXISTS spaces;

DROP FUNCTION IF EXISTS fn_bump_tasks_updated_at_on_level_rename;
DROP FUNCTION IF EXISTS fn_nullify_tasks_on_priority_delete;
DROP FUNCTION IF EXISTS fn_nullify_tasks_on_effort_delete;
DROP FUNCTION IF EXISTS fn_prevent_status_delete_if_tasks_exist;

DROP TYPE IF EXISTS overdue_action;
DROP TYPE IF EXISTS activity_action;
DROP TYPE IF EXISTS activity_entity_type;
DROP TYPE IF EXISTS stored_relation_kind;
DROP TYPE IF EXISTS recurrence_type;
DROP TYPE IF EXISTS auth_token_kind;
DROP TYPE IF EXISTS space_role;
DROP TYPE IF EXISTS status_category;

DROP EXTENSION IF EXISTS pg_trgm;
