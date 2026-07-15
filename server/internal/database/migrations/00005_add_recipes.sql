-- +goose Up

-- PostgreSQL does not support removing individual enum values. The Down migration
-- therefore leaves this value in place, and IF NOT EXISTS allows this migration to
-- be applied again after a rollback.
ALTER TYPE activity_entity_type ADD VALUE IF NOT EXISTS 'recipe';

CREATE TABLE recipes (
    id            BIGSERIAL      NOT NULL PRIMARY KEY,
    space_slug    TEXT           NOT NULL,
    name          TEXT           NOT NULL,
    description   TEXT           NOT NULL DEFAULT '',
    yield_amount  NUMERIC(12, 4),
    yield_unit    TEXT,
    prep_minutes  INTEGER,
    cook_minutes  INTEGER,
    created_at    TIMESTAMPTZ    NOT NULL,
    updated_at    TIMESTAMPTZ    NOT NULL,
    FOREIGN KEY (space_slug) REFERENCES spaces (slug) ON DELETE CASCADE ON UPDATE CASCADE,
    UNIQUE (id, space_slug),
    CHECK (btrim(name) <> ''),
    CHECK ((yield_amount IS NULL) = (yield_unit IS NULL)),
    CHECK (yield_amount IS NULL OR yield_amount > 0),
    CHECK (yield_unit IS NULL OR btrim(yield_unit) <> ''),
    CHECK (prep_minutes IS NULL OR prep_minutes >= 0),
    CHECK (cook_minutes IS NULL OR cook_minutes >= 0)
);

CREATE INDEX idx_recipes_space_updated
    ON recipes (space_slug, updated_at DESC, id DESC);
CREATE INDEX idx_recipes_name_trgm
    ON recipes
    USING GIN (lower(name) gin_trgm_ops);

CREATE TABLE recipe_ingredient_sections (
    id        BIGSERIAL NOT NULL PRIMARY KEY,
    recipe_id BIGINT    NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    title     TEXT      NOT NULL DEFAULT '',
    position  INTEGER   NOT NULL,
    UNIQUE (recipe_id, position) DEFERRABLE INITIALLY DEFERRED,
    CHECK (position >= 0)
);

CREATE TABLE recipe_ingredients (
    id           BIGSERIAL     NOT NULL PRIMARY KEY,
    section_id   BIGINT        NOT NULL REFERENCES recipe_ingredient_sections (id) ON DELETE CASCADE,
    position     INTEGER       NOT NULL,
    quantity     NUMERIC(12, 4),
    quantity_max NUMERIC(12, 4),
    unit         TEXT          NOT NULL DEFAULT '',
    item         TEXT          NOT NULL,
    UNIQUE (section_id, position) DEFERRABLE INITIALLY DEFERRED,
    CHECK (position >= 0),
    CHECK (btrim(item) <> ''),
    CHECK (quantity IS NULL OR quantity > 0),
    CHECK (
        quantity_max IS NULL
        OR (quantity IS NOT NULL AND quantity_max >= quantity)
    )
);

CREATE INDEX idx_recipe_ingredients_item_trgm
    ON recipe_ingredients
    USING GIN (lower(item) gin_trgm_ops);

CREATE TABLE recipe_instruction_sections (
    id        BIGSERIAL NOT NULL PRIMARY KEY,
    recipe_id BIGINT    NOT NULL REFERENCES recipes (id) ON DELETE CASCADE,
    title     TEXT      NOT NULL DEFAULT '',
    position  INTEGER   NOT NULL,
    UNIQUE (recipe_id, position) DEFERRABLE INITIALLY DEFERRED,
    CHECK (position >= 0)
);

CREATE TABLE recipe_steps (
    id         BIGSERIAL NOT NULL PRIMARY KEY,
    section_id BIGINT    NOT NULL REFERENCES recipe_instruction_sections (id) ON DELETE CASCADE,
    position   INTEGER   NOT NULL,
    body       TEXT      NOT NULL,
    UNIQUE (section_id, position) DEFERRABLE INITIALLY DEFERRED,
    CHECK (position >= 0),
    CHECK (btrim(body) <> '')
);

ALTER TABLE tags
ADD CONSTRAINT tags_id_space_unique UNIQUE (id, space_slug);

CREATE TABLE recipe_tags (
    recipe_id  BIGINT      NOT NULL,
    tag_id     BIGINT      NOT NULL,
    space_slug TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (recipe_id, tag_id),
    FOREIGN KEY (recipe_id, space_slug)
        REFERENCES recipes (id, space_slug)
        ON DELETE CASCADE
        ON UPDATE CASCADE,
    FOREIGN KEY (tag_id, space_slug)
        REFERENCES tags (id, space_slug)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

CREATE INDEX idx_recipe_tags_tag
    ON recipe_tags (tag_id, recipe_id);

-- +goose Down

DROP TABLE recipe_tags;
ALTER TABLE tags DROP CONSTRAINT tags_id_space_unique;
DROP TABLE recipe_steps;
DROP TABLE recipe_instruction_sections;
DROP TABLE recipe_ingredients;
DROP TABLE recipe_ingredient_sections;
DROP TABLE recipes;

-- activity_entity_type.recipe intentionally remains; PostgreSQL cannot remove
-- individual enum values without rebuilding the type.
