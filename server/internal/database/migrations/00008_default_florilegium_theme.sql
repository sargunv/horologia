-- +goose Up

ALTER TABLE users
ALTER COLUMN appearance_light_theme SET DEFAULT 'florilegium';

-- +goose Down

ALTER TABLE users
ALTER COLUMN appearance_light_theme SET DEFAULT 'light';
