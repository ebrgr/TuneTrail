-- +goose Up
-- Shared by ISSUE-217 and ISSUE-218; ISSUE-195 folds this into the consolidated schema.
CREATE TABLE metadata_sources (
    id INTEGER PRIMARY KEY,
    source TEXT NOT NULL CHECK(length(source)>0),
    instance TEXT NOT NULL CHECK(length(instance)>0),
    UNIQUE(source,instance)
);
CREATE TABLE descriptive_metadata (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE CASCADE,
    release_id INTEGER REFERENCES releases(id) ON DELETE CASCADE,
    source_id INTEGER NOT NULL REFERENCES metadata_sources(id),
    kind TEXT NOT NULL CHECK(kind IN ('genre','tag','recording_year','release_date','artist_origin_country','isrc','bpm','first_release_date','first_recording_date')),
    value TEXT NOT NULL CHECK(length(trim(value))>0),
    observed_at TEXT NOT NULL CHECK(length(observed_at)>0),
    CHECK((artist_id IS NOT NULL)+(occurrence_id IS NOT NULL)+(release_id IS NOT NULL)=1),
    CHECK(kind NOT IN ('recording_year','release_date','artist_origin_country','isrc','bpm','first_release_date','first_recording_date')
       OR (kind IN ('recording_year','isrc','bpm','first_recording_date') AND occurrence_id IS NOT NULL)
       OR (kind IN ('release_date','first_release_date') AND release_id IS NOT NULL)
       OR (kind='artist_origin_country' AND artist_id IS NOT NULL))
);
CREATE UNIQUE INDEX idx_descriptive_artist ON descriptive_metadata(profile_id,artist_id,kind,source_id,value) WHERE artist_id IS NOT NULL;
CREATE UNIQUE INDEX idx_descriptive_occurrence ON descriptive_metadata(profile_id,occurrence_id,kind,source_id,value) WHERE occurrence_id IS NOT NULL;
CREATE UNIQUE INDEX idx_descriptive_release ON descriptive_metadata(profile_id,release_id,kind,source_id,value) WHERE release_id IS NOT NULL;
CREATE INDEX idx_descriptive_artist_fk ON descriptive_metadata(artist_id) WHERE artist_id IS NOT NULL;
CREATE INDEX idx_descriptive_occurrence_fk ON descriptive_metadata(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_descriptive_release_fk ON descriptive_metadata(release_id) WHERE release_id IS NOT NULL;

-- A profile may only claim an entity supported by one of its listens.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_scope_insert BEFORE INSERT ON descriptive_metadata
WHEN NOT EXISTS (
    SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id
    WHERE l.profile_id=NEW.profile_id AND (
        NEW.occurrence_id=o.id OR NEW.release_id=o.release_id OR
        EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=o.id AND a.artist_id=NEW.artist_id) OR
        EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=o.id AND c.artist_id=NEW.artist_id)))
BEGIN
    SELECT RAISE(ABORT,'Descriptive metadata has no supporting listen');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_scope_update BEFORE UPDATE OF profile_id,artist_id,occurrence_id,release_id ON descriptive_metadata
BEGIN
    SELECT RAISE(ABORT,'Descriptive metadata binding is immutable');
END;
-- +goose StatementEnd
-- Prune only when the profile no longer has a supporting listen for the entity.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_listen_delete AFTER DELETE ON listens
BEGIN
    DELETE FROM descriptive_metadata WHERE profile_id=OLD.profile_id
      AND (occurrence_id=OLD.track_id OR release_id=(SELECT release_id FROM occurrences WHERE id=OLD.track_id)
           OR artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.track_id UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.track_id))
      AND NOT EXISTS (SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=OLD.profile_id
        AND (descriptive_metadata.occurrence_id=o.id OR descriptive_metadata.release_id=o.release_id
          OR EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=o.id AND a.artist_id=descriptive_metadata.artist_id)
          OR EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=o.id AND c.artist_id=descriptive_metadata.artist_id)));
END;
-- +goose StatementEnd
-- A cascading occurrence deletion removes its listens after the parent has disappeared.
-- Prune affected release and artist observations while the relationships are still available.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
 DELETE FROM descriptive_metadata WHERE
 (release_id=OLD.release_id OR artist_id IN (
   SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id
   UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.id))
 AND NOT EXISTS (SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id
   WHERE l.profile_id=descriptive_metadata.profile_id AND o.id<>OLD.id AND
   (descriptive_metadata.release_id=o.release_id
    OR EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=o.id AND a.artist_id=descriptive_metadata.artist_id)
    OR EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=o.id AND c.artist_id=descriptive_metadata.artist_id)));
END;
-- +goose StatementEnd
-- Legacy tracks own artist links, which cascade before the occurrence is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_track_delete BEFORE DELETE ON tracks
BEGIN
 DELETE FROM descriptive_metadata WHERE artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id)
 AND NOT EXISTS (SELECT 1 FROM listens l WHERE l.profile_id=descriptive_metadata.profile_id AND l.track_id<>OLD.id
   AND (EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=l.track_id AND a.artist_id=descriptive_metadata.artist_id)
     OR EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=l.track_id AND c.artist_id=descriptive_metadata.artist_id)));
END;
-- +goose StatementEnd
-- +goose Down
CREATE TEMP TABLE descriptive_metadata_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO descriptive_metadata_downgrade_guard VALUES (1);
