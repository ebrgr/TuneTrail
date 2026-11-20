-- +goose Up
ALTER TABLE profile_ingest_flags ADD COLUMN occurrence_write INTEGER NOT NULL DEFAULT 0
    CHECK(occurrence_write IN (0,1));

ALTER TABLE ingest_events ADD COLUMN resolver_variant TEXT NOT NULL DEFAULT 'v9'
    CHECK(resolver_variant IN ('v9','occurrence'));

ALTER TABLE occurrences ADD COLUMN presented_title TEXT;

CREATE TABLE occurrence_credits (
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK(position>=0),
    name TEXT NOT NULL CHECK(length(name)>0),
    artist_id INTEGER REFERENCES artists(id) ON DELETE SET NULL,
    PRIMARY KEY(occurrence_id,position)
);

INSERT INTO event_origins(origin,description) VALUES
    ('listenbrainz_api','ListenBrainz HTTP submission'),
    ('listenbrainz_import','ListenBrainz archive import'),
    ('lastfm_import','Last.fm history import'),
    ('spotify_import','Spotify history import'),
    ('maloja_import','Maloja history import'),
    ('native_import','TuneTrail or Koito history import');

-- Repair only linked active receipts affected by earlier catalog mutations.
CREATE TEMP TABLE occurrence_link_repairs AS
SELECT e.profile_id,e.receipt_id,e.resolution_version+1 AS version,e.duplicate_decision,
       l.track_id AS occurrence_id,o.recording_id,
       json_object('before',json_object('occurrence_id',e.occurrence_id,'recording_id',e.recording_id,'identity_status',e.identity_status),
                   'after',json_object('occurrence_id',l.track_id,'recording_id',o.recording_id,'identity_status','resolved')) AS evidence
FROM ingest_events e
JOIN listens l ON (l.profile_id,l.event_id)=(e.profile_id,e.event_id)
JOIN occurrences o ON o.id=l.track_id
WHERE e.mode='active' AND e.state IN ('accepted','duplicate')
  AND (e.occurrence_id IS NOT l.track_id OR e.recording_id IS NOT o.recording_id OR e.identity_status IS NOT 'resolved');
CREATE UNIQUE INDEX temp.idx_occurrence_link_repairs ON occurrence_link_repairs(receipt_id);

INSERT INTO ingest_revisions(profile_id,receipt_id,resolution_version,author,decision,reason_codes,evidence,rule_version,created_at)
SELECT profile_id,receipt_id,version,'migration',duplicate_decision,'["migration_link_repair"]',evidence,'occurrence-link-repair-v1',unixepoch()
FROM occurrence_link_repairs;

UPDATE ingest_events AS e
SET occurrence_id=r.occurrence_id,recording_id=r.recording_id,identity_status='resolved',resolution_version=r.version
FROM occurrence_link_repairs r
WHERE e.receipt_id=r.receipt_id;

DROP TABLE occurrence_link_repairs;

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_flag_one_way BEFORE UPDATE OF occurrence_write,active,profile_id ON profile_ingest_flags
WHEN (OLD.occurrence_write=1 AND (NEW.occurrence_write=0 OR NEW.active=0 OR NEW.profile_id IS NOT OLD.profile_id))
  OR (NEW.occurrence_write=1 AND NEW.active=0)
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing cannot be disabled');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_flag_insert BEFORE INSERT ON profile_ingest_flags
WHEN NEW.occurrence_write=1 AND NEW.active=0
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing requires active ingestion');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_flag_delete BEFORE DELETE ON profile_ingest_flags
WHEN OLD.occurrence_write=1 AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=OLD.profile_id)
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing cannot be disabled by deleting flags');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_resolver_variant_immutable BEFORE UPDATE OF resolver_variant ON ingest_events
WHEN NEW.resolver_variant IS NOT OLD.resolver_variant
BEGIN
    SELECT RAISE(ABORT,'Receipt resolver variant is immutable');
END;
-- +goose StatementEnd

DROP TRIGGER trg_identity_legacy_occurrence;
-- +goose StatementBegin
CREATE TRIGGER trg_identity_occurrence_insert BEFORE INSERT ON occurrences
WHEN (NEW.legacy_track_id IS NULL AND
      (NEW.release_id IS NOT NULL OR coalesce(length(trim(NEW.presented_title)),0)=0))
  OR (NEW.legacy_track_id IS NOT NULL AND NOT EXISTS (
      SELECT 1 FROM tracks WHERE id=NEW.legacy_track_id AND release_id=NEW.release_id))
BEGIN
    SELECT RAISE(ABORT,'Occurrence requires a matching track or a title without a release');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_identity_occurrence_bridge_update BEFORE UPDATE OF legacy_track_id,release_id ON occurrences
WHEN NEW.legacy_track_id IS NOT OLD.legacy_track_id
  OR (NEW.legacy_track_id IS NULL AND NEW.release_id IS NOT NULL)
BEGIN
    SELECT RAISE(ABORT,'Cannot change occurrence bridge or assign a release without a track');
END;
-- +goose StatementEnd

DROP TRIGGER trg_identity_legacy_listen_insert;
-- +goose StatementBegin
CREATE TRIGGER trg_identity_listen_origin_insert BEFORE INSERT ON listens
WHEN NOT EXISTS (SELECT 1 FROM event_origins WHERE origin=NEW.origin)
BEGIN
    SELECT RAISE(ABORT,'Unregistered listen origin');
END;
-- +goose StatementEnd

DROP TRIGGER trg_identity_legacy_listen_update;
-- +goose StatementBegin
CREATE TRIGGER trg_identity_listen_origin_immutable BEFORE UPDATE OF origin ON listens
WHEN NEW.origin IS NOT OLD.origin
BEGIN
    SELECT RAISE(ABORT,'Listen origin is immutable');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_identity_track_recording_mbid BEFORE UPDATE OF musicbrainz_id ON tracks
WHEN NEW.musicbrainz_id IS NOT OLD.musicbrainz_id AND EXISTS (
    SELECT 1 FROM occurrences o JOIN recordings r ON r.id=o.recording_id
    WHERE o.legacy_track_id=NEW.id AND (r.mbid_source='validated' OR EXISTS (
        SELECT 1 FROM occurrences other WHERE other.recording_id=r.id AND other.id<>o.id))
)
BEGIN
    SELECT RAISE(ABORT,'Cannot change MBID of a shared or validated recording through a track');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_identity_listen_occurrence_guard BEFORE UPDATE OF track_id ON listens
WHEN NEW.track_id IS NOT OLD.track_id AND EXISTS (
    SELECT 1 FROM ingest_events WHERE profile_id=OLD.profile_id AND event_id=OLD.event_id AND resolver_variant='occurrence')
BEGIN
    SELECT RAISE(ABORT,'Cannot move an occurrence-resolved listen');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_identity_occurrence_recording_guard BEFORE UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id AND EXISTS (
    SELECT 1 FROM listens l JOIN ingest_events e ON (e.profile_id,e.event_id)=(l.profile_id,l.event_id)
    WHERE l.track_id=OLD.id AND e.resolver_variant='occurrence')
BEGIN
    SELECT RAISE(ABORT,'Cannot reassign a linked occurrence recording');
END;
-- +goose StatementEnd

-- A foreign-key SET NULL can precede the deletion of the associated listen.
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_link_insert BEFORE INSERT ON ingest_events
WHEN NEW.mode='active' AND NEW.state IN ('accepted','duplicate') AND (NEW.identity_status IS NOT 'resolved' OR NOT EXISTS (
    SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id
    WHERE (l.profile_id,l.event_id)=(NEW.profile_id,NEW.event_id)
      AND (NEW.occurrence_id IS NULL OR NEW.occurrence_id=l.track_id)
      AND (NEW.recording_id IS NULL OR NEW.recording_id=o.recording_id)))
BEGIN
    SELECT RAISE(ABORT,'Inconsistent resolved ingest link');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_link_update BEFORE UPDATE OF state,identity_status,profile_id,event_id,occurrence_id,recording_id ON ingest_events
WHEN NEW.mode='active' AND NEW.state IN ('accepted','duplicate') AND (NEW.identity_status IS NOT 'resolved' OR NOT EXISTS (
    SELECT 1 FROM listens l LEFT JOIN occurrences o ON o.id=l.track_id
    WHERE (l.profile_id,l.event_id)=(NEW.profile_id,NEW.event_id)
      AND (NEW.occurrence_id IS NULL OR NEW.occurrence_id=l.track_id)
      AND (NEW.recording_id IS NULL OR NEW.recording_id=o.recording_id
           OR (o.id IS NULL AND NEW.occurrence_id IS NULL AND NEW.recording_id IS OLD.recording_id))))
BEGIN
    SELECT RAISE(ABORT,'Inconsistent resolved ingest link');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_resolution_version BEFORE UPDATE OF resolution_version,state,identity_status,duplicate_decision,event_id,occurrence_id,recording_id ON ingest_events
WHEN NEW.mode='active' AND (NEW.resolution_version<OLD.resolution_version OR (
    NEW.resolution_version=OLD.resolution_version AND (OLD.state IN ('accepted','duplicate') OR NEW.state IN ('accepted','duplicate'))
    AND (NEW.state IS NOT OLD.state OR NEW.identity_status IS NOT OLD.identity_status
         OR NEW.duplicate_decision IS NOT OLD.duplicate_decision OR NEW.event_id IS NOT OLD.event_id
         OR (NEW.occurrence_id IS NOT NULL AND NEW.occurrence_id IS NOT OLD.occurrence_id)
         OR (NEW.recording_id IS NOT NULL AND NEW.recording_id IS NOT OLD.recording_id))))
BEGIN
    SELECT RAISE(ABORT,'Ingest decision requires a newer resolution version');
END;
-- +goose StatementEnd

-- Keep surviving v9 receipts aligned when the legacy merge moves a listen.
-- Ignored conflicts do not fire this trigger and are deleted by the merge.
-- +goose StatementBegin
CREATE TRIGGER trg_identity_listen_sync_v9 AFTER UPDATE OF track_id ON listens
WHEN NEW.track_id IS NOT OLD.track_id
BEGIN
    INSERT INTO ingest_revisions(profile_id,receipt_id,resolution_version,author,decision,reason_codes,evidence,rule_version,created_at)
    SELECT e.profile_id,e.receipt_id,e.resolution_version+1,'automatic',e.duplicate_decision,
           '["legacy_listen_moved"]',
           json_object('before',json_object('occurrence_id',e.occurrence_id,'recording_id',e.recording_id),
                       'after',json_object('occurrence_id',NEW.track_id,'recording_id',o.recording_id)),
           'occurrence-link-repair-v1',unixepoch()
    FROM ingest_events e JOIN occurrences o ON o.id=NEW.track_id
    WHERE (e.profile_id,e.event_id)=(NEW.profile_id,NEW.event_id)
      AND e.resolver_variant='v9' AND e.state IN ('accepted','duplicate');

    UPDATE ingest_events
    SET occurrence_id=NEW.track_id,
        recording_id=(SELECT recording_id FROM occurrences WHERE id=NEW.track_id),
        resolution_version=resolution_version+1
    WHERE (profile_id,event_id)=(NEW.profile_id,NEW.event_id)
      AND resolver_variant='v9' AND state IN ('accepted','duplicate');
END;
-- +goose StatementEnd

-- +goose Down
-- Restore the pre-upgrade backup to return to v9.
CREATE TEMP TABLE occurrence_ingest_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO occurrence_ingest_downgrade_guard VALUES (1);
