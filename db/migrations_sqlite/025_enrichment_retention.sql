-- +goose Up
-- Declarations belong to the profile's catalog, not to an individual receipt.
CREATE TABLE enrichment_recording_claims_new (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    instance TEXT NOT NULL,
    mbid TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY(profile_id,occurrence_id,instance,mbid,source)
);
INSERT INTO enrichment_recording_claims_new
SELECT profile_id,occurrence_id,instance,mbid,source,min(observed_at)
FROM enrichment_recording_claims GROUP BY profile_id,occurrence_id,instance,mbid,source;
DROP TABLE enrichment_recording_claims;
ALTER TABLE enrichment_recording_claims_new RENAME TO enrichment_recording_claims;

CREATE TABLE enrichment_artist_claims_new (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    instance TEXT NOT NULL,
    mbid TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY(profile_id,artist_id,instance,mbid,source)
);
INSERT INTO enrichment_artist_claims_new
SELECT profile_id,artist_id,instance,mbid,source,min(observed_at)
FROM enrichment_artist_claims GROUP BY profile_id,artist_id,instance,mbid,source;
DROP TABLE enrichment_artist_claims;
ALTER TABLE enrichment_artist_claims_new RENAME TO enrichment_artist_claims;

-- Move claims before the occurrence's foreign-key cascades remove them.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
    INSERT OR IGNORE INTO enrichment_recording_claims(profile_id,occurrence_id,instance,mbid,source,observed_at)
    SELECT c.profile_id,l.track_id,c.instance,c.mbid,c.source,c.observed_at
    FROM enrichment_recording_claims c JOIN occurrences other ON other.recording_id=OLD.recording_id AND other.id<>OLD.id
    JOIN listens l ON l.track_id=other.id AND l.profile_id=c.profile_id
    WHERE c.occurrence_id=OLD.id AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=c.profile_id);
END;
-- +goose StatementEnd

-- A profile's declarations are removed when its last related listen is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_listen_delete AFTER DELETE ON listens
BEGIN
    -- A recording can span occurrences; retain the declaration on remaining profile listens.
    INSERT OR IGNORE INTO enrichment_recording_claims(profile_id,occurrence_id,instance,mbid,source,observed_at)
    SELECT c.profile_id,l.track_id,c.instance,c.mbid,c.source,c.observed_at
    FROM enrichment_recording_claims c JOIN occurrences old_occurrence ON old_occurrence.id=c.occurrence_id
    JOIN occurrences other ON other.recording_id=old_occurrence.recording_id AND other.id<>old_occurrence.id
    JOIN listens l ON l.track_id=other.id AND l.profile_id=c.profile_id
    WHERE c.profile_id=OLD.profile_id AND c.occurrence_id=OLD.track_id
      AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=OLD.profile_id)
      AND NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id);
    DELETE FROM enrichment_recording_claims WHERE profile_id=OLD.profile_id AND occurrence_id=OLD.track_id
      AND NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id);
    DELETE FROM enrichment_artist_claims WHERE profile_id=OLD.profile_id
      AND NOT EXISTS (SELECT 1 FROM listens l WHERE l.profile_id=OLD.profile_id AND (
        EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=l.track_id AND a.artist_id=enrichment_artist_claims.artist_id)
        OR EXISTS (SELECT 1 FROM occurrence_credits a WHERE a.occurrence_id=l.track_id AND a.artist_id=enrichment_artist_claims.artist_id)));
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE enrichment_retention_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO enrichment_retention_downgrade_guard VALUES (1);
