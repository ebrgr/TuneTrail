-- +goose Up
CREATE TABLE archive_identity (
    singleton INTEGER PRIMARY KEY CHECK(singleton=1),
    namespace TEXT NOT NULL UNIQUE
);
INSERT INTO archive_identity VALUES (1,lower(hex(randomblob(16))));

CREATE TABLE archive_catalog (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    namespace TEXT NOT NULL,
    kind TEXT NOT NULL,
    source_id INTEGER NOT NULL,
    local_id INTEGER NOT NULL,
    trusted INTEGER NOT NULL CHECK(trusted IN (0,1)),
    PRIMARY KEY(profile_id,namespace,kind,source_id)
) WITHOUT ROWID;

CREATE INDEX idx_archive_catalog_trusted ON archive_catalog(namespace,kind,source_id) WHERE trusted=1;
CREATE INDEX idx_archive_catalog_local ON archive_catalog(kind,local_id);

-- Artists and releases can reuse a deleted row ID. Remove stale associations first.
-- +goose StatementBegin
CREATE TRIGGER trg_archive_artist_delete AFTER DELETE ON artists
BEGIN
    DELETE FROM archive_catalog WHERE kind='artists' AND local_id=OLD.id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_archive_release_delete AFTER DELETE ON releases
BEGIN
    DELETE FROM archive_catalog WHERE kind='releases' AND local_id=OLD.id;
END;
-- +goose StatementEnd

CREATE TABLE archive_events (
    profile_id INTEGER NOT NULL,
    namespace TEXT NOT NULL,
    source_event_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    origin TEXT NOT NULL,
    provenance TEXT NOT NULL CHECK(json_valid(provenance)),
    PRIMARY KEY(profile_id,namespace,source_event_id),
    UNIQUE(profile_id,event_id),
    FOREIGN KEY(profile_id,event_id) REFERENCES listens(profile_id,event_id) ON DELETE CASCADE
);
INSERT INTO event_origins VALUES ('archive_restore','Versioned TuneTrail history restore');

-- Legacy admission retains its existing deduplication rule. Native events use event_id.
DROP INDEX idx_listens_legacy_identity;
CREATE UNIQUE INDEX idx_listens_legacy_identity ON listens(profile_id,track_id,listened_at) WHERE origin='legacy';

CREATE TABLE recording_moves (
    source_track INTEGER NOT NULL,
    destination_track INTEGER NOT NULL,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    original_recording INTEGER NOT NULL,
    merged_recording INTEGER NOT NULL,
    stage INTEGER NOT NULL CHECK(stage IN (0,1,2)),
    PRIMARY KEY(source_track,destination_track,occurrence_id)
);
CREATE INDEX idx_recording_moves_occurrence ON recording_moves(occurrence_id);

DROP TRIGGER trg_identity_occurrence_recording_guard;
-- +goose StatementBegin
CREATE TRIGGER trg_identity_occurrence_recording_guard BEFORE UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id
  AND EXISTS (
    SELECT 1 FROM listens l JOIN ingest_events e ON (e.profile_id,e.event_id)=(l.profile_id,l.event_id)
    WHERE l.track_id=OLD.id AND e.resolver_variant='occurrence')
  AND NOT EXISTS (
    SELECT 1 FROM recording_moves m WHERE m.occurrence_id=OLD.id AND (
      (m.stage=0 AND m.original_recording=OLD.recording_id AND m.merged_recording=NEW.recording_id)
      OR (m.stage=2 AND m.merged_recording=OLD.recording_id AND m.original_recording=NEW.recording_id)))
BEGIN
    SELECT RAISE(ABORT,'Cannot reassign a linked occurrence recording');
END;
-- +goose StatementEnd

-- Record catalog reconciliation in every affected accepted receipt's revision history.
-- +goose StatementBegin
CREATE TRIGGER trg_recording_move_revisions AFTER UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id AND EXISTS (
    SELECT 1 FROM recording_moves m WHERE m.occurrence_id=NEW.id AND m.stage IN (0,2))
BEGIN
    INSERT INTO ingest_revisions(profile_id,receipt_id,resolution_version,author,decision,reason_codes,evidence,rule_version,created_at)
    SELECT e.profile_id,e.receipt_id,e.resolution_version+1,'catalog',e.duplicate_decision,
           '["recording_reconciled"]',json_object('before',OLD.recording_id,'after',NEW.recording_id),
           'recording-reconciliation-v1',unixepoch()
    FROM ingest_events e JOIN listens l ON (e.profile_id,e.event_id)=(l.profile_id,l.event_id)
    WHERE l.track_id=NEW.id AND e.mode='active' AND e.state IN ('accepted','duplicate');

    UPDATE ingest_events SET recording_id=NEW.recording_id,resolution_version=resolution_version+1
    WHERE (profile_id,event_id) IN (SELECT profile_id,event_id FROM listens WHERE track_id=NEW.id)
      AND mode='active' AND state IN ('accepted','duplicate');

    SELECT RAISE(ABORT,'Missing link revision') WHERE EXISTS (
        SELECT 1 FROM ingest_events e JOIN listens l ON (e.profile_id,e.event_id)=(l.profile_id,l.event_id)
        WHERE l.track_id=NEW.id AND e.mode='active' AND e.state IN ('accepted','duplicate')
          AND NOT EXISTS (SELECT 1 FROM ingest_revisions r
              WHERE (r.profile_id,r.receipt_id,r.resolution_version)=(e.profile_id,e.receipt_id,e.resolution_version)));
END;
-- +goose StatementEnd

-- +goose Down
-- Restored history requires the archive identity and provenance tables.
CREATE TEMP TABLE archive_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO archive_downgrade_guard VALUES (1);
