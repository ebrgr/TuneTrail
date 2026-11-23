-- +goose Up
-- ISSUE-202: an optional avatar background chosen from the fixed palette in internal/models/avatar_color.go.
-- NULL keeps the theme colour. The CHECK guards only the format (#rrggbb in lower case); the application
-- accepts nothing outside the palette, so no free-form value ever reaches the CSS.
ALTER TABLE listening_profiles ADD COLUMN avatar_color TEXT
    CHECK (avatar_color IS NULL OR (length(avatar_color) = 7 AND avatar_color GLOB '#[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'));

-- +goose Down
CREATE TEMP TABLE avatar_color_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO avatar_color_downgrade_guard VALUES (1);
