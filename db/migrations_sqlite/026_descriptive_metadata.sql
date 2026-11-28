-- +goose Up
-- Shared by ISSUE-217 and ISSUE-218; ISSUE-195 folds this into the consolidated schema.
CREATE TABLE metadata_sources (
    id INTEGER PRIMARY KEY,
    source TEXT NOT NULL CHECK(length(source)>0),
    instance TEXT NOT NULL CHECK(length(instance)>0),
    UNIQUE(source,instance)
);
-- A transient marker preserves retry after partial detail failure across process restarts.
CREATE TABLE metadata_retries (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    source_id INTEGER NOT NULL REFERENCES metadata_sources(id),
    PRIMARY KEY(profile_id,occurrence_id,source_id)
) WITHOUT ROWID;
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

-- Entity-specific lookups start with the bounded entity relation, then probe the profile/track index.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_scope_insert BEFORE INSERT ON descriptive_metadata
BEGIN
    SELECT CASE WHEN NEW.occurrence_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM listens WHERE profile_id=NEW.profile_id AND track_id=NEW.occurrence_id
    ) THEN RAISE(ABORT,'Descriptive metadata has no supporting listen') END;
    SELECT CASE WHEN NEW.release_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM occurrences o INDEXED BY idx_occurrences_release
        CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE o.release_id=NEW.release_id AND l.profile_id=NEW.profile_id AND l.track_id=o.id
    ) THEN RAISE(ABORT,'Descriptive metadata has no supporting listen') END;
    SELECT CASE WHEN NEW.artist_id IS NOT NULL
        AND NOT EXISTS (SELECT 1 FROM artist_tracks a CROSS JOIN listens l INDEXED BY idx_listens_profile_track
            WHERE a.artist_id=NEW.artist_id AND l.profile_id=NEW.profile_id AND l.track_id=a.track_id)
        AND NOT EXISTS (SELECT 1 FROM occurrence_credits c INDEXED BY idx_occurrence_credits_artist CROSS JOIN listens l INDEXED BY idx_listens_profile_track
            WHERE c.artist_id=NEW.artist_id AND l.profile_id=NEW.profile_id AND l.track_id=c.occurrence_id)
    THEN RAISE(ABORT,'Descriptive metadata has no supporting listen') END;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_scope_update BEFORE UPDATE OF profile_id,artist_id,occurrence_id,release_id ON descriptive_metadata
BEGIN
    SELECT RAISE(ABORT,'Descriptive metadata binding is immutable');
END;
-- +goose StatementEnd

-- Bounded profile-first probes near the changed track and at the start avoid repeated scans of shared entities.
-- Each deletion is limited to the entity affected by this listen.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_listen_delete AFTER DELETE ON listens
BEGIN
    DELETE FROM metadata_retries WHERE profile_id=OLD.profile_id AND occurrence_id=OLD.track_id
      AND NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id);
    DELETE FROM descriptive_metadata WHERE profile_id=OLD.profile_id AND occurrence_id=OLD.track_id
      AND NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id);
    DELETE FROM descriptive_metadata WHERE profile_id=OLD.profile_id
      AND release_id=(SELECT release_id FROM occurrences WHERE id=OLD.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id AND track_id>=OLD.track_id LIMIT 16) recent
        CROSS JOIN occurrences o WHERE o.id=recent.track_id AND o.release_id=descriptive_metadata.release_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id LIMIT 16) recent
        CROSS JOIN occurrences o WHERE o.id=recent.track_id AND o.release_id=descriptive_metadata.release_id)
      AND NOT EXISTS (SELECT 1 FROM occurrences o INDEXED BY idx_occurrences_release CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE o.release_id=descriptive_metadata.release_id AND l.profile_id=OLD.profile_id AND l.track_id=o.id);
    DELETE FROM descriptive_metadata WHERE profile_id=OLD.profile_id
      AND artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.track_id UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id AND track_id>=OLD.track_id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id AND track_id>=OLD.track_id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=OLD.profile_id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM artist_tracks a CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE a.artist_id=descriptive_metadata.artist_id AND l.profile_id=OLD.profile_id AND l.track_id=a.track_id)
      AND NOT EXISTS (SELECT 1 FROM occurrence_credits c INDEXED BY idx_occurrence_credits_artist CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE c.artist_id=descriptive_metadata.artist_id AND l.profile_id=OLD.profile_id AND l.track_id=c.occurrence_id);
END;
-- +goose StatementEnd

-- Parent cascades can remove listens after their occurrence is gone.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
    DELETE FROM descriptive_metadata WHERE release_id=OLD.release_id
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id>=OLD.id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrences o WHERE o.id=recent.track_id AND o.release_id=descriptive_metadata.release_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrences o WHERE o.id=recent.track_id AND o.release_id=descriptive_metadata.release_id)
      AND NOT EXISTS (SELECT 1 FROM occurrences o INDEXED BY idx_occurrences_release CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE o.release_id=descriptive_metadata.release_id AND o.id<>OLD.id AND l.profile_id=descriptive_metadata.profile_id AND l.track_id=o.id);
    DELETE FROM descriptive_metadata WHERE artist_id IN
      (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id>=OLD.id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id>=OLD.id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM artist_tracks a CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id<>OLD.id AND l.profile_id=descriptive_metadata.profile_id AND l.track_id=a.track_id)
      AND NOT EXISTS (SELECT 1 FROM occurrence_credits c INDEXED BY idx_occurrence_credits_artist CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE c.artist_id=descriptive_metadata.artist_id AND c.occurrence_id<>OLD.id AND l.profile_id=descriptive_metadata.profile_id AND l.track_id=c.occurrence_id);
END;
-- +goose StatementEnd

-- Legacy tracks own artist links that cascade before the occurrence is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_descriptive_track_delete BEFORE DELETE ON tracks
BEGIN
    DELETE FROM descriptive_metadata WHERE artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id>=OLD.id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN artist_tracks a WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id=recent.track_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id>=OLD.id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM (SELECT track_id FROM listens WHERE profile_id=descriptive_metadata.profile_id AND track_id<>OLD.id LIMIT 16) recent
        CROSS JOIN occurrence_credits c INDEXED BY sqlite_autoindex_occurrence_credits_1 WHERE c.occurrence_id=recent.track_id AND c.artist_id=descriptive_metadata.artist_id)
      AND NOT EXISTS (SELECT 1 FROM artist_tracks a CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE a.artist_id=descriptive_metadata.artist_id AND a.track_id<>OLD.id AND l.profile_id=descriptive_metadata.profile_id AND l.track_id=a.track_id)
      AND NOT EXISTS (SELECT 1 FROM occurrence_credits c INDEXED BY idx_occurrence_credits_artist CROSS JOIN listens l INDEXED BY idx_listens_profile_track
        WHERE c.artist_id=descriptive_metadata.artist_id AND c.occurrence_id<>OLD.id AND l.profile_id=descriptive_metadata.profile_id AND l.track_id=c.occurrence_id);
END;
-- +goose StatementEnd
-- +goose Down
CREATE TEMP TABLE descriptive_metadata_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO descriptive_metadata_downgrade_guard VALUES (1);
