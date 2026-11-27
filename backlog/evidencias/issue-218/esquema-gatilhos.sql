CREATE TABLE artist_tracks (
    artist_id  INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    track_id   INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    is_primary INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (artist_id, track_id)
);
CREATE TABLE "tracks" (
    id             INTEGER PRIMARY KEY,
    musicbrainz_id TEXT UNIQUE,
    release_id     INTEGER NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    duration       INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE occurrences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recording_id INTEGER NOT NULL REFERENCES recordings(id),
    release_id INTEGER REFERENCES releases(id) ON DELETE SET NULL,
    legacy_track_id INTEGER UNIQUE CHECK(legacy_track_id IS NULL OR legacy_track_id=id),
    track_mbid TEXT UNIQUE,
    disc_number INTEGER CHECK(disc_number>0),
    track_number INTEGER CHECK(track_number>0)
, presented_title TEXT);
CREATE TABLE "listens" (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    track_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    listened_at INTEGER NOT NULL,
    user_id INTEGER,
    client TEXT NOT NULL DEFAULT '',
    event_id TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))) CHECK(length(event_id)>0),
    origin TEXT NOT NULL DEFAULT 'legacy' REFERENCES event_origins(origin) ON DELETE RESTRICT, review_distinct INTEGER NOT NULL DEFAULT 0 CHECK(review_distinct IN (0,1)), duration_played INTEGER
    CHECK (duration_played IS NULL OR (typeof(duration_played) = 'integer' AND duration_played >= 0)),
    UNIQUE(profile_id,event_id)
);
CREATE TABLE occurrence_credits (
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK(position>=0),
    name TEXT NOT NULL CHECK(length(name)>0),
    artist_id INTEGER REFERENCES artists(id) ON DELETE SET NULL,
    PRIMARY KEY(occurrence_id,position)
);
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
CREATE INDEX idx_artist_tracks_track_id    ON artist_tracks(track_id);
CREATE INDEX idx_artist_tracks_artist_id   ON artist_tracks(artist_id);
CREATE INDEX idx_tracks_release_id ON tracks(release_id);
CREATE INDEX idx_occurrences_recording ON occurrences(recording_id);
CREATE INDEX idx_occurrences_release ON occurrences(release_id);
CREATE INDEX idx_listens_profile_listened_at ON listens(profile_id,listened_at);
CREATE INDEX idx_listens_listened_at ON listens(listened_at);
CREATE INDEX idx_listens_track_id ON listens(track_id);
CREATE INDEX idx_listens_track_id_listened_at ON listens(track_id,listened_at);
CREATE INDEX idx_listens_user_id ON listens(user_id);
CREATE INDEX idx_listens_event_cursor ON listens(profile_id,listened_at,event_id);
CREATE INDEX idx_occurrence_credits_artist ON occurrence_credits(artist_id);
CREATE INDEX idx_listens_profile_track ON listens(profile_id,track_id);
CREATE UNIQUE INDEX idx_descriptive_artist ON descriptive_metadata(profile_id,artist_id,kind,source_id,value) WHERE artist_id IS NOT NULL;
CREATE UNIQUE INDEX idx_descriptive_occurrence ON descriptive_metadata(profile_id,occurrence_id,kind,source_id,value) WHERE occurrence_id IS NOT NULL;
CREATE UNIQUE INDEX idx_descriptive_release ON descriptive_metadata(profile_id,release_id,kind,source_id,value) WHERE release_id IS NOT NULL;
CREATE INDEX idx_descriptive_artist_fk ON descriptive_metadata(artist_id) WHERE artist_id IS NOT NULL;
CREATE INDEX idx_descriptive_occurrence_fk ON descriptive_metadata(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_descriptive_release_fk ON descriptive_metadata(release_id) WHERE release_id IS NOT NULL;
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
CREATE TRIGGER trg_descriptive_scope_update BEFORE UPDATE OF profile_id,artist_id,occurrence_id,release_id ON descriptive_metadata
BEGIN
    SELECT RAISE(ABORT,'Descriptive metadata binding is immutable');
END;
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
CREATE TRIGGER trg_descriptive_track_delete BEFORE DELETE ON tracks
BEGIN
 DELETE FROM descriptive_metadata WHERE artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id)
 AND NOT EXISTS (SELECT 1 FROM listens l WHERE l.profile_id=descriptive_metadata.profile_id AND l.track_id<>OLD.id
   AND (EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=l.track_id AND a.artist_id=descriptive_metadata.artist_id)
     OR EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=l.track_id AND c.artist_id=descriptive_metadata.artist_id)));
END;
