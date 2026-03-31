-- +goose Up
ALTER TYPE activity_entity_type ADD VALUE IF NOT EXISTS 'user';

-- +goose Down
-- Enum values cannot be removed in PostgreSQL; intentional no-op.
