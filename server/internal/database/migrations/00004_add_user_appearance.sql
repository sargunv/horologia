-- +goose Up

ALTER TABLE users
ADD COLUMN appearance_mode TEXT NOT NULL DEFAULT 'system',
ADD COLUMN appearance_light_theme TEXT NOT NULL DEFAULT 'light',
ADD COLUMN appearance_dark_theme TEXT NOT NULL DEFAULT 'dark',
ADD CONSTRAINT users_appearance_mode_chk
    CHECK (appearance_mode IN ('system', 'light', 'dark')),
ADD CONSTRAINT users_appearance_light_theme_chk
    CHECK (length(appearance_light_theme) BETWEEN 1 AND 100),
ADD CONSTRAINT users_appearance_dark_theme_chk
    CHECK (length(appearance_dark_theme) BETWEEN 1 AND 100);

-- +goose Down

ALTER TABLE users
DROP CONSTRAINT users_appearance_dark_theme_chk,
DROP CONSTRAINT users_appearance_light_theme_chk,
DROP CONSTRAINT users_appearance_mode_chk,
DROP COLUMN appearance_dark_theme,
DROP COLUMN appearance_light_theme,
DROP COLUMN appearance_mode;
