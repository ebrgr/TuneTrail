-- +goose Up
-- Remove the old retention paths before rebuilding the active catalog.
DROP TRIGGER trg_metadata_scope_insert;
DROP TRIGGER trg_metadata_scope_update;
DROP TRIGGER trg_metadata_recording_reconcile;
DROP TRIGGER trg_enrichment_occurrence_delete;
DROP TRIGGER trg_enrichment_listen_delete;
DROP TRIGGER trg_enrichment_artist_listen_delete;
DROP TRIGGER trg_enrichment_artist_occurrence_delete;
DROP TRIGGER trg_enrichment_artist_bridge_delete;
ALTER TABLE metadata_values RENAME TO metadata_values_v27;
DROP INDEX idx_metadata_artist;
DROP INDEX idx_metadata_recording;
DROP INDEX idx_metadata_recording_read;
DROP INDEX idx_metadata_release;
DROP INDEX idx_metadata_artist_fk;
DROP INDEX idx_metadata_recording_fk;
DROP INDEX idx_metadata_release_fk;
DROP INDEX idx_metadata_occurrence_fk;
DROP INDEX idx_metadata_event_fk;

CREATE TABLE metadata_values (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    recording_id INTEGER REFERENCES recordings(id) ON DELETE CASCADE,
    release_id INTEGER REFERENCES releases(id) ON DELETE CASCADE,
    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES listens(event_id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('mbid','genre','tag','recording_year','release_date','artist_origin_country')),
    value TEXT NOT NULL CHECK(length(value)>0),
    instance TEXT NOT NULL CHECK(length(instance)>0),
    source TEXT NOT NULL CHECK(length(source)>0),
    observed_at TEXT NOT NULL CHECK(length(observed_at)>0),
    CHECK((artist_id IS NOT NULL)+(recording_id IS NOT NULL)+(release_id IS NOT NULL)=1),
    CHECK((recording_id IS NULL)=(occurrence_id IS NULL)),
    CHECK(kind NOT IN ('recording_year','release_date','artist_origin_country')
        OR (kind='recording_year' AND recording_id IS NOT NULL)
        OR (kind='release_date' AND release_id IS NOT NULL)
        OR (kind='artist_origin_country' AND artist_id IS NOT NULL))
);
CREATE UNIQUE INDEX idx_metadata_artist
    ON metadata_values(profile_id,artist_id,kind,instance,value,source,event_id) WHERE artist_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_recording
    ON metadata_values(profile_id,occurrence_id,kind,instance,value,source,event_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_read
    ON metadata_values(profile_id,recording_id,kind,instance,value) WHERE recording_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_release
    ON metadata_values(profile_id,release_id,kind,instance,value,source,event_id) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_artist_fk ON metadata_values(artist_id) WHERE artist_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_fk ON metadata_values(recording_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_release_fk ON metadata_values(release_id) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_occurrence_fk ON metadata_values(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_metadata_event_fk ON metadata_values(event_id);

-- Defensive guards also apply to future providers and archive restores.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_scope_insert BEFORE INSERT ON metadata_values
BEGIN
    SELECT CASE WHEN NEW.recording_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM occurrences WHERE id=NEW.occurrence_id AND recording_id=NEW.recording_id
    ) THEN RAISE(ABORT,'Metadata recording does not match occurrence') END;
    SELECT CASE WHEN NEW.event_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id
        WHERE l.event_id=NEW.event_id AND l.profile_id=NEW.profile_id AND (
            NEW.occurrence_id=o.id OR NEW.release_id=o.release_id
            OR EXISTS (SELECT 1 FROM artist_tracks WHERE track_id=o.id AND artist_id=NEW.artist_id)
            OR EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=o.id AND artist_id=NEW.artist_id))
    ) THEN RAISE(ABORT,'Metadata event does not support entity and profile') END;
END;
-- +goose StatementEnd

-- Only the recording reconciliation trigger changes an existing entity binding.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_scope_update BEFORE UPDATE ON metadata_values
WHEN NEW.profile_id IS NOT OLD.profile_id OR NEW.artist_id IS NOT OLD.artist_id
  OR NEW.release_id IS NOT OLD.release_id OR NEW.occurrence_id IS NOT OLD.occurrence_id
  OR NEW.event_id IS NOT OLD.event_id
  OR (NEW.recording_id IS NOT OLD.recording_id AND NOT EXISTS (
        SELECT 1 FROM occurrences WHERE id=NEW.occurrence_id AND recording_id=NEW.recording_id))
BEGIN
    SELECT RAISE(ABORT,'Metadata observation binding is immutable');
END;
-- +goose StatementEnd


-- Keep unverifiable v27 values out of all active readers. Historical event IDs
-- are not copied into quarantine because they may belong to another profile.
CREATE TABLE metadata_unverified_legacy (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    recording_id INTEGER REFERENCES recordings(id) ON DELETE CASCADE,
    release_id INTEGER REFERENCES releases(id) ON DELETE CASCADE,
    occurrence_id INTEGER,
    kind TEXT NOT NULL,
    value TEXT NOT NULL,
    instance TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    reason TEXT NOT NULL CHECK(reason IN ('missing_event','unverified_event'))
);
CREATE INDEX idx_metadata_unverified_profile ON metadata_unverified_legacy(profile_id);

-- Test the original event only. Other related listens do not prove historical provenance.
INSERT INTO metadata_values
SELECT m.* FROM metadata_values_v27 m
JOIN listens l ON l.event_id=m.event_id AND l.profile_id=m.profile_id
JOIN occurrences o ON o.id=l.track_id
WHERE (m.recording_id=o.recording_id AND m.occurrence_id=o.id)
   OR m.release_id=o.release_id
   OR EXISTS (SELECT 1 FROM artist_tracks WHERE track_id=o.id AND artist_id=m.artist_id)
   OR EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=o.id AND artist_id=m.artist_id);

INSERT INTO metadata_unverified_legacy
SELECT m.id,m.profile_id,m.artist_id,m.recording_id,m.release_id,m.occurrence_id,m.kind,m.value,m.instance,m.source,m.observed_at,
       CASE WHEN m.event_id IS NULL THEN 'missing_event' ELSE 'unverified_event' END
FROM metadata_values_v27 m WHERE NOT EXISTS (SELECT 1 FROM metadata_values v WHERE v.id=m.id);
DROP TABLE metadata_values_v27;

-- Occurrences stay put when a recording merge is applied or undone.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_recording_reconcile AFTER UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id
BEGIN
    UPDATE metadata_values SET recording_id=NEW.recording_id WHERE occurrence_id=NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE metadata_origin_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO metadata_origin_downgrade_guard VALUES (1);
