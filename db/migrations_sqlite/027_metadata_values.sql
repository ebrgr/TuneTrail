-- +goose Up
-- Each observation belongs to one catalog entity and one listening profile.
CREATE TABLE metadata_values (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    recording_id INTEGER REFERENCES recordings(id) ON DELETE CASCADE,
    release_id INTEGER REFERENCES releases(id) ON DELETE CASCADE,
    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE CASCADE,
    event_id TEXT REFERENCES listens(event_id) ON DELETE SET NULL,
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
    ON metadata_values(profile_id,artist_id,kind,instance,value,source) WHERE artist_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_recording
    ON metadata_values(profile_id,occurrence_id,kind,instance,value,source) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_read
    ON metadata_values(profile_id,recording_id,kind,instance,value) WHERE recording_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_release
    ON metadata_values(profile_id,release_id,kind,instance,value,source) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_artist_fk ON metadata_values(artist_id) WHERE artist_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_fk ON metadata_values(recording_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_release_fk ON metadata_values(release_id) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_occurrence_fk ON metadata_values(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_metadata_event_fk ON metadata_values(event_id) WHERE event_id IS NOT NULL;
CREATE INDEX idx_ingest_received_recording ON ingest_events(profile_id,recording_id)
    WHERE state IN ('accepted','duplicate') AND
    (coalesce(json_extract(original_snapshot,'$.recording_mbid'),'')<>'' OR coalesce(json_extract(original_snapshot,'$.mapped_recording_mbid'),'')<>'');

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
  OR (NEW.event_id IS NOT OLD.event_id AND NEW.event_id IS NOT NULL)
  OR (NEW.recording_id IS NOT OLD.recording_id AND NOT EXISTS (
        SELECT 1 FROM occurrences WHERE id=NEW.occurrence_id AND recording_id=NEW.recording_id))
BEGIN
    SELECT RAISE(ABORT,'Metadata observation binding is immutable');
END;
-- +goose StatementEnd

INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,kind,value,instance,source,observed_at)
SELECT c.profile_id,o.recording_id,c.occurrence_id,'mbid',c.mbid,c.instance,c.source,c.observed_at
FROM enrichment_recording_claims c JOIN occurrences o ON o.id=c.occurrence_id;
INSERT INTO metadata_values(profile_id,artist_id,kind,value,instance,source,observed_at)
SELECT profile_id,artist_id,'mbid',mbid,instance,source,observed_at FROM enrichment_artist_claims;

DROP TRIGGER trg_enrichment_occurrence_delete;
DROP TRIGGER trg_enrichment_listen_delete;
DROP TRIGGER trg_enrichment_artist_listen_delete;
DROP TRIGGER trg_enrichment_artist_occurrence_delete;
DROP TRIGGER trg_enrichment_artist_bridge_delete;
DROP TABLE enrichment_recording_claims;
DROP TABLE enrichment_artist_claims;

-- Occurrences stay put when a recording merge is applied or undone.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_recording_reconcile AFTER UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id
BEGIN
    UPDATE metadata_values SET recording_id=NEW.recording_id WHERE occurrence_id=NEW.id;
END;
-- +goose StatementEnd

-- Move claims before the occurrence's foreign-key cascades remove them.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
    INSERT OR IGNORE INTO metadata_values(profile_id,recording_id,occurrence_id,kind,value,instance,source,observed_at)
    SELECT c.profile_id,other.recording_id,other.id,c.kind,c.value,c.instance,c.source,c.observed_at
    FROM metadata_values c JOIN occurrences other ON other.recording_id=OLD.recording_id AND other.id<>OLD.id
    WHERE c.occurrence_id=OLD.id AND EXISTS (SELECT 1 FROM listens WHERE profile_id=c.profile_id AND track_id=other.id) AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=c.profile_id);
END;
-- +goose StatementEnd

-- A profile's declarations are removed when its last related listen is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_listen_delete AFTER DELETE ON listens
WHEN NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id)
BEGIN
    -- A recording can span occurrences; retain the declaration on remaining profile listens.
    INSERT OR IGNORE INTO metadata_values(profile_id,recording_id,occurrence_id,kind,value,instance,source,observed_at)
    SELECT c.profile_id,other.recording_id,other.id,c.kind,c.value,c.instance,c.source,c.observed_at
    FROM metadata_values c JOIN occurrences old_occurrence ON old_occurrence.id=c.occurrence_id
    JOIN occurrences other ON other.recording_id=old_occurrence.recording_id AND other.id<>old_occurrence.id
    WHERE c.profile_id=OLD.profile_id AND c.occurrence_id=OLD.track_id
      AND EXISTS (SELECT 1 FROM listens WHERE profile_id=c.profile_id AND track_id=other.id)
      AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=OLD.profile_id);
    DELETE FROM metadata_values WHERE profile_id=OLD.profile_id AND occurrence_id=OLD.track_id;
    DELETE FROM metadata_values
    WHERE profile_id=OLD.profile_id AND release_id=(SELECT release_id FROM occurrences WHERE id=OLD.track_id)
      AND NOT EXISTS (
        SELECT 1 FROM occurrences o CROSS JOIN listens l
        WHERE o.release_id=metadata_values.release_id AND l.profile_id=OLD.profile_id AND l.track_id=o.id
      );
END;
-- +goose StatementEnd

-- Only the affected occurrence's artists can have lost a supporting listen.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_listen_delete AFTER DELETE ON listens
WHEN NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id)
BEGIN
    DELETE FROM metadata_values
    WHERE profile_id=OLD.profile_id AND artist_id IN (
        SELECT artist_id FROM artist_tracks WHERE track_id=OLD.track_id
        UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.track_id AND artist_id IS NOT NULL
    )
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id
          AND l.profile_id=OLD.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id
          AND l.profile_id=OLD.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- Clean up before occurrence credits can disappear in the foreign-key cascade.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
    DELETE FROM metadata_values
    WHERE release_id=OLD.release_id AND profile_id IN (SELECT profile_id FROM listens WHERE track_id=OLD.id)
      AND NOT EXISTS (
        SELECT 1 FROM occurrences o CROSS JOIN listens l
        WHERE o.release_id=OLD.release_id AND o.id<>OLD.id
          AND l.profile_id=metadata_values.profile_id AND l.track_id=o.id
      );
    DELETE FROM metadata_values
    WHERE profile_id IN (SELECT profile_id FROM listens WHERE track_id=OLD.id)
    AND artist_id IN (
        SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id
        UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.id AND artist_id IS NOT NULL
    )
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id AND a.track_id<>OLD.id
          AND l.profile_id=metadata_values.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id AND a.occurrence_id<>OLD.id
          AND l.profile_id=metadata_values.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- A legacy track loses its artist relations before its occurrence is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_bridge_delete BEFORE DELETE ON tracks
BEGIN
    DELETE FROM metadata_values
    WHERE profile_id IN (SELECT profile_id FROM listens WHERE track_id=OLD.id)
    AND artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id)
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id AND a.track_id<>OLD.id
          AND l.profile_id=metadata_values.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=metadata_values.artist_id AND a.occurrence_id<>OLD.id
          AND l.profile_id=metadata_values.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE metadata_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO metadata_downgrade_guard VALUES (1);
