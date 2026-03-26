-- +goose Up

-- Enum types
CREATE TYPE status_category AS ENUM ('initial', 'intermediate', 'completion');
CREATE TYPE space_role AS ENUM ('admin', 'member', 'viewer');
CREATE TYPE auth_token_kind AS ENUM ('session', 'api');
CREATE TYPE recurrence_type AS ENUM ('one_off', 'completion_based', 'fixed_non_accumulating', 'fixed_accumulating', 'on_dependency');
CREATE TYPE stored_relation_kind AS ENUM ('parent', 'blocks', 'relates_to', 'duplicates', 'triggers', 'spawns');

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
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Task effort levels
CREATE TABLE task_effort_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Task priority levels
CREATE TABLE task_priority_levels (
    space_slug TEXT    NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    name       TEXT    NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (space_slug, name),
    UNIQUE (space_slug, position) DEFERRABLE INITIALLY DEFERRED
);

-- Tasks
CREATE TABLE tasks (
    id              BIGSERIAL       NOT NULL PRIMARY KEY,
    space_slug      TEXT            NOT NULL,
    title           TEXT            NOT NULL,
    description     TEXT            NOT NULL DEFAULT '',
    status_name     TEXT            NOT NULL,
    effort_name     TEXT,
    priority_name   TEXT,
    due_at          DATE,
    due_tz          TEXT,
    recurrence_type recurrence_type NOT NULL DEFAULT 'one_off',
    recurrence_rule TEXT,
    last_completed_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL,
    updated_at      TIMESTAMPTZ     NOT NULL,
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
    id         BIGSERIAL       NOT NULL PRIMARY KEY,
    user_id    BIGINT          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT            NOT NULL UNIQUE,
    name       TEXT            NOT NULL DEFAULT '',
    kind       auth_token_kind NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ     NOT NULL
);

CREATE INDEX idx_auth_tokens_user ON auth_tokens (user_id);
CREATE INDEX idx_auth_tokens_expires ON auth_tokens (expires_at)
    WHERE expires_at IS NOT NULL;

-- Space members
CREATE TABLE space_members (
    space_slug TEXT        NOT NULL REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       space_role  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (space_slug, user_id)
);

CREATE INDEX idx_space_members_user ON space_members (user_id);

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
    source_task_id BIGINT              NOT NULL,
    target_task_id BIGINT              NOT NULL,
    space_slug     TEXT                NOT NULL,
    kind           stored_relation_kind NOT NULL,
    created_at     TIMESTAMPTZ         NOT NULL,
    PRIMARY KEY (source_task_id, target_task_id, kind),
    FOREIGN KEY (source_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (target_task_id, space_slug) REFERENCES tasks (id, space_slug) ON DELETE CASCADE ON UPDATE CASCADE,
    CHECK (source_task_id != target_task_id),
    CHECK (kind NOT IN ('relates_to', 'duplicates') OR source_task_id < target_task_id)
);

CREATE INDEX idx_task_relations_target ON task_relations (target_task_id);
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

-- +goose Down

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

DROP FUNCTION IF EXISTS fn_nullify_tasks_on_priority_delete;
DROP FUNCTION IF EXISTS fn_nullify_tasks_on_effort_delete;
DROP FUNCTION IF EXISTS fn_prevent_status_delete_if_tasks_exist;

DROP TYPE IF EXISTS stored_relation_kind;
DROP TYPE IF EXISTS recurrence_type;
DROP TYPE IF EXISTS auth_token_kind;
DROP TYPE IF EXISTS space_role;
DROP TYPE IF EXISTS status_category;
