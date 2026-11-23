-- +goose Up
-- Existing listens have no measured playback time. Keep their catalog fallback.
ALTER TABLE listens ADD COLUMN duration_played INTEGER
    CHECK (duration_played IS NULL OR (typeof(duration_played) = 'integer' AND duration_played >= 0));

-- +goose Down
CREATE TEMP TABLE duration_played_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO duration_played_downgrade_guard VALUES (1);
