-- +goose Up

ALTER TABLE users
ALTER COLUMN appearance_light_theme SET DEFAULT 'frutiger-aero';

-- +goose Down

ALTER TABLE users
ALTER COLUMN appearance_light_theme SET DEFAULT 'light';
