-- +goose Up
-- Keep foreign keys enabled. Save and remove children before replacing their parents.
CREATE TEMP TABLE identity_assertion (name TEXT PRIMARY KEY, ok INTEGER NOT NULL CHECK(ok=1));
CREATE TEMP TABLE old_ingest_events AS SELECT * FROM ingest_events;
CREATE TEMP TABLE old_ingest_event_details AS SELECT * FROM ingest_event_details;
CREATE TEMP TABLE old_ingest_observations AS SELECT * FROM ingest_observations;
CREATE TEMP TABLE old_ingest_revisions AS SELECT * FROM ingest_revisions;
CREATE TEMP TABLE old_outbox AS SELECT * FROM outbox;
CREATE TEMP TABLE old_outbox_details AS SELECT * FROM outbox_details;
DROP TABLE ingest_event_details;
DROP TABLE ingest_observations;
DROP TABLE ingest_revisions;
DROP TABLE outbox_details;
DROP TABLE ingest_events;
DROP TABLE outbox;

-- Both tables are guarded in the source schema; unexpected rows must abort.
INSERT INTO identity_assertion VALUES ('inactive queues',
    NOT EXISTS (SELECT 1 FROM old_ingest_revisions) AND NOT EXISTS (SELECT 1 FROM old_outbox));

CREATE TABLE recordings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    musicbrainz_id TEXT UNIQUE,
    mbid_source TEXT NOT NULL DEFAULT 'legacy_declared'
        CHECK(mbid_source IN ('legacy_declared','validated'))
);
CREATE TABLE occurrences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recording_id INTEGER NOT NULL REFERENCES recordings(id),
    release_id INTEGER REFERENCES releases(id) ON DELETE SET NULL,
    legacy_track_id INTEGER UNIQUE CHECK(legacy_track_id IS NULL OR legacy_track_id=id),
    track_mbid TEXT UNIQUE,
    disc_number INTEGER CHECK(disc_number>0),
    track_number INTEGER CHECK(track_number>0)
);
CREATE INDEX idx_occurrences_recording ON occurrences(recording_id);
CREATE INDEX idx_occurrences_release ON occurrences(release_id);

-- Allocate recording IDs independently from the legacy track IDs.
CREATE TEMP TABLE old_track_map AS
SELECT id AS track_id, row_number() OVER (ORDER BY id) AS recording_id, release_id, musicbrainz_id
FROM tracks;
INSERT INTO recordings (id,musicbrainz_id)
SELECT recording_id,musicbrainz_id FROM old_track_map;
INSERT INTO occurrences (id,recording_id,release_id,legacy_track_id)
SELECT track_id,recording_id,release_id,track_id FROM old_track_map;
INSERT INTO identity_assertion VALUES ('track mapping',
    (SELECT count(*) FROM tracks)=(SELECT count(*) FROM occurrences)
    AND (SELECT count(*) FROM tracks)=(SELECT count(*) FROM recordings)
    AND NOT EXISTS (
        SELECT 1 FROM old_track_map m
        LEFT JOIN occurrences o ON o.id=m.track_id
        LEFT JOIN recordings r ON r.id=o.recording_id
        WHERE o.legacy_track_id IS NOT m.track_id OR o.release_id IS NOT m.release_id
           OR r.id IS NOT m.recording_id OR r.musicbrainz_id IS NOT m.musicbrainz_id
    ));

CREATE TEMP TABLE old_listens AS SELECT rowid AS legacy_rowid, * FROM listens;
-- The activation migration can register origins without rebuilding listens.
CREATE TABLE event_origins (
    origin TEXT NOT NULL PRIMARY KEY CHECK(length(origin)>0),
    description TEXT NOT NULL
);
INSERT INTO event_origins VALUES ('legacy','Legacy listen or backfilled history');
CREATE TABLE listens_new (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    track_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    listened_at INTEGER NOT NULL,
    user_id INTEGER,
    client TEXT NOT NULL DEFAULT '',
    event_id TEXT NOT NULL PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))) CHECK(length(event_id)>0),
    origin TEXT NOT NULL DEFAULT 'legacy' REFERENCES event_origins(origin) ON DELETE RESTRICT,
    UNIQUE(profile_id,event_id)
);
INSERT INTO listens_new (rowid,profile_id,track_id,listened_at,user_id,client,origin)
SELECT legacy_rowid,profile_id,track_id,listened_at,user_id,client,'legacy' FROM old_listens;
DROP TABLE listens;
ALTER TABLE listens_new RENAME TO listens;
CREATE UNIQUE INDEX idx_listens_legacy_identity ON listens(profile_id,track_id,listened_at);
CREATE INDEX idx_listens_profile_listened_at ON listens(profile_id,listened_at);
CREATE INDEX idx_listens_listened_at ON listens(listened_at);
CREATE INDEX idx_listens_track_id ON listens(track_id);
CREATE INDEX idx_listens_track_id_listened_at ON listens(track_id,listened_at);
CREATE INDEX idx_listens_user_id ON listens(user_id);
CREATE INDEX idx_listens_event_cursor ON listens(profile_id,listened_at,event_id);

CREATE TABLE ingest_events (
    receipt_id TEXT PRIMARY KEY NOT NULL CHECK(length(receipt_id)>0),
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    source_instance_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL CHECK(schema_version>0),
    source_type TEXT NOT NULL CHECK(length(source_type)>0),
    adapter_version TEXT NOT NULL CHECK(length(adapter_version)>0),
    batch_id TEXT NOT NULL CHECK(length(batch_id)>0),
    item_ordinal INTEGER NOT NULL CHECK(item_ordinal>=0),
    operation TEXT NOT NULL CHECK(length(operation)>0),
    received_at INTEGER NOT NULL,
    external_namespace TEXT CHECK(external_namespace IS NULL OR length(external_namespace)>0),
    external_event_id TEXT CHECK(external_event_id IS NULL OR length(external_event_id)>0),
    external_id_trusted INTEGER NOT NULL DEFAULT 0 CHECK(external_id_trusted IN (0,1)),
    client_idempotency_key TEXT CHECK(client_idempotency_key IS NULL OR length(client_idempotency_key)>0),
    batch_digest TEXT,
    file_digest TEXT CHECK(file_digest IS NULL OR (length(file_digest)=64 AND file_digest NOT GLOB '*[^0-9a-f]*')),
    file_format TEXT,
    format_version TEXT,
    listened_at TEXT,
    original_time TEXT,
    time_precision TEXT NOT NULL CHECK(time_precision IN ('unknown','second','millisecond','microsecond','nanosecond','minute','estimated')),
    time_semantics TEXT NOT NULL CHECK(time_semantics IN ('start','end','unknown')),
    original_offset TEXT,
    uncertainty_ms INTEGER CHECK(uncertainty_ms>=0),
    client_name TEXT,
    client_version TEXT,
    original_snapshot TEXT NOT NULL CHECK(json_valid(original_snapshot) AND json_type(original_snapshot)='object'),
    normalized_snapshot TEXT CHECK(normalized_snapshot IS NULL OR (json_valid(normalized_snapshot) AND json_type(normalized_snapshot)='object')),
    normalizer_version TEXT,
    snapshot_digest TEXT NOT NULL CHECK(length(snapshot_digest)>0),
    digest_version TEXT NOT NULL CHECK(length(digest_version)>0),
    state TEXT NOT NULL DEFAULT 'received' CHECK(state IN ('received','pending_identity','pending_duplicate','invalid','accepted','duplicate','retryable')),
    identity_status TEXT NOT NULL DEFAULT 'unresolved' CHECK(identity_status IN ('unresolved','pending','invalid','resolved')),
    duplicate_decision TEXT CHECK(duplicate_decision IN ('same','different','review')),
    reason_codes TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(reason_codes) AND json_type(reason_codes)='array'),
    evidence TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(evidence) AND json_type(evidence)='object'),
    rule_version TEXT,
    resolution_version INTEGER NOT NULL DEFAULT 0 CHECK(resolution_version>=0),
    resolved_at INTEGER,
    event_id TEXT,
    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE SET NULL,
    recording_id INTEGER REFERENCES recordings(id) ON DELETE SET NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count>=0),
    next_attempt_at INTEGER,
    lease_owner TEXT,
    lease_until INTEGER,
    work_version INTEGER NOT NULL DEFAULT 0 CHECK(work_version>=0),
    closed_at INTEGER,
    UNIQUE(profile_id,receipt_id),
    FOREIGN KEY(profile_id,event_id) REFERENCES listens(profile_id,event_id) ON DELETE CASCADE,
    CHECK((state IN ('accepted','duplicate'))=(event_id IS NOT NULL)),
    FOREIGN KEY(profile_id,source_instance_id) REFERENCES integrations(profile_id,id) ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED,
    CHECK(external_id_trusted=0 OR (external_namespace IS NOT NULL AND external_event_id IS NOT NULL)),
    CHECK((external_namespace IS NULL)=(external_event_id IS NULL)),
    CHECK(client_idempotency_key IS NULL OR (batch_digest IS NOT NULL AND length(batch_digest)>0)),
    CHECK((file_digest IS NULL AND file_format IS NULL AND format_version IS NULL) OR
          (file_digest IS NOT NULL AND file_format IS NOT NULL AND length(file_format)>0 AND format_version IS NOT NULL AND length(format_version)>0)),
    CHECK((normalized_snapshot IS NULL)=(normalizer_version IS NULL)),
    CHECK((lease_owner IS NULL)=(lease_until IS NULL)),
    CHECK(closed_at IS NULL OR state IN ('invalid','accepted','duplicate')),
    CHECK(state!='invalid' OR (rule_version IS NOT NULL AND json_array_length(reason_codes)>0))
);
CREATE UNIQUE INDEX idx_ingest_external_key ON ingest_events(profile_id,source_instance_id,external_namespace,external_event_id) WHERE external_id_trusted=1;
CREATE UNIQUE INDEX idx_ingest_client_key ON ingest_events(profile_id,source_instance_id,operation,client_idempotency_key,item_ordinal) WHERE client_idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX idx_ingest_file_key ON ingest_events(profile_id,source_instance_id,file_format,format_version,file_digest,item_ordinal) WHERE file_digest IS NOT NULL;
CREATE INDEX idx_ingest_queue ON ingest_events(state,next_attempt_at,received_at) WHERE closed_at IS NULL;
CREATE INDEX idx_ingest_event ON ingest_events(profile_id,event_id);
CREATE INDEX idx_ingest_occurrence ON ingest_events(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_ingest_recording ON ingest_events(recording_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_ingest_profile_time ON ingest_events(profile_id,listened_at);

CREATE TABLE ingest_event_details (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL,
    receipt_id TEXT NOT NULL,
    resolution_version INTEGER NOT NULL CHECK(resolution_version>=0),
    kind TEXT NOT NULL CHECK(length(kind)>0),
    payload TEXT NOT NULL CHECK(json_valid(payload)),
    closed_at INTEGER,
    protected INTEGER NOT NULL DEFAULT 0 CHECK(protected IN (0,1)),
    FOREIGN KEY(profile_id,receipt_id) REFERENCES ingest_events(profile_id,receipt_id) ON DELETE CASCADE
);
CREATE INDEX idx_ingest_details_purge ON ingest_event_details(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0;
CREATE INDEX idx_ingest_details_receipt ON ingest_event_details(profile_id,receipt_id);

CREATE TABLE ingest_observations (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)>0),
    profile_id INTEGER NOT NULL,
    receipt_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL CHECK(schema_version>0),
    snapshot_digest TEXT NOT NULL CHECK(length(snapshot_digest)>0),
    digest_version TEXT NOT NULL CHECK(length(digest_version)>0),
    original_snapshot TEXT NOT NULL CHECK(json_valid(original_snapshot) AND json_type(original_snapshot)='object'),
    received_at INTEGER NOT NULL,
    closed_at INTEGER,
    protected INTEGER NOT NULL DEFAULT 0 CHECK(protected IN (0,1)),
    UNIQUE(profile_id,receipt_id,digest_version,snapshot_digest),
    FOREIGN KEY(profile_id,receipt_id) REFERENCES ingest_events(profile_id,receipt_id) ON DELETE CASCADE
);
CREATE INDEX idx_ingest_observations_purge ON ingest_observations(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0;
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_observation_immutable BEFORE UPDATE ON ingest_observations
WHEN NEW.id IS NOT OLD.id OR NEW.profile_id IS NOT OLD.profile_id OR NEW.receipt_id IS NOT OLD.receipt_id
 OR NEW.schema_version IS NOT OLD.schema_version OR NEW.snapshot_digest IS NOT OLD.snapshot_digest
 OR NEW.digest_version IS NOT OLD.digest_version OR NEW.original_snapshot IS NOT OLD.original_snapshot OR NEW.received_at IS NOT OLD.received_at
BEGIN
    SELECT RAISE(ABORT,'Observed variants are immutable');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_snapshot_immutable BEFORE UPDATE ON ingest_events
WHEN NEW.receipt_id IS NOT OLD.receipt_id OR NEW.profile_id IS NOT OLD.profile_id
 OR NEW.source_instance_id IS NOT OLD.source_instance_id OR NEW.original_snapshot IS NOT OLD.original_snapshot
 OR NEW.schema_version IS NOT OLD.schema_version OR NEW.snapshot_digest IS NOT OLD.snapshot_digest OR NEW.digest_version IS NOT OLD.digest_version
BEGIN
    SELECT RAISE(ABORT,'Receipt identity and observed snapshot are immutable');
END;
-- +goose StatementEnd

CREATE TABLE ingest_revisions (
    profile_id INTEGER NOT NULL,
    receipt_id TEXT NOT NULL,
    resolution_version INTEGER NOT NULL CHECK(resolution_version>0),
    author TEXT NOT NULL CHECK(length(author)>0),
    decision TEXT NOT NULL CHECK(decision IN ('same','different','review','invalid')),
    reason_codes TEXT NOT NULL CHECK(json_valid(reason_codes) AND json_type(reason_codes)='array' AND json_array_length(reason_codes)>0),
    evidence TEXT NOT NULL CHECK(json_valid(evidence) AND json_type(evidence)='object'),
    rule_version TEXT NOT NULL CHECK(length(rule_version)>0),
    created_at INTEGER NOT NULL,
    PRIMARY KEY(profile_id,receipt_id,resolution_version),
    FOREIGN KEY(profile_id,receipt_id) REFERENCES ingest_events(profile_id,receipt_id) ON DELETE CASCADE
);
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_revision_inactive BEFORE INSERT ON ingest_revisions
BEGIN
    SELECT RAISE(ABORT,'Receipt resolution requires pipeline activation');
END;
-- +goose StatementEnd

CREATE TABLE outbox (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)>0),
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    destination_id TEXT NOT NULL,
    event_id TEXT NOT NULL CHECK(length(event_id)>0),
    operation TEXT NOT NULL CHECK(length(operation)>0),
    delivery_key TEXT NOT NULL CHECK(length(delivery_key)>0),
    state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','retryable','failed','sent','discarded')),
    created_at INTEGER NOT NULL,
    next_attempt_at INTEGER,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count>=0),
    lease_owner TEXT,
    lease_until INTEGER,
    work_version INTEGER NOT NULL DEFAULT 0 CHECK(work_version>=0),
    closed_at INTEGER,
    result TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(result) AND json_type(result)='object'),
    UNIQUE(profile_id,id),
    FOREIGN KEY(profile_id,event_id) REFERENCES listens(profile_id,event_id) ON DELETE CASCADE,
    UNIQUE(profile_id,destination_id,event_id,operation),
    UNIQUE(profile_id,destination_id,delivery_key),
    FOREIGN KEY(profile_id,destination_id) REFERENCES integrations(profile_id,id) ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED,
    CHECK((lease_owner IS NULL)=(lease_until IS NULL)),
    CHECK((state IN ('sent','discarded'))=(closed_at IS NOT NULL))
);
CREATE INDEX idx_outbox_queue ON outbox(state,next_attempt_at,created_at) WHERE closed_at IS NULL;
CREATE INDEX idx_outbox_event ON outbox(profile_id,event_id);
-- +goose StatementBegin
CREATE TRIGGER trg_outbox_inactive BEFORE INSERT ON outbox
BEGIN
    SELECT RAISE(ABORT,'Outbox requires pipeline activation');
END;
-- +goose StatementEnd
CREATE TABLE outbox_details (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL,
    outbox_id TEXT NOT NULL,
    payload TEXT NOT NULL CHECK(json_valid(payload)),
    closed_at INTEGER,
    protected INTEGER NOT NULL DEFAULT 0 CHECK(protected IN (0,1)),
    FOREIGN KEY(profile_id,outbox_id) REFERENCES outbox(profile_id,id) ON DELETE CASCADE
);
CREATE INDEX idx_outbox_details_purge ON outbox_details(closed_at,id) WHERE closed_at IS NOT NULL AND protected=0;
CREATE INDEX idx_outbox_details_parent ON outbox_details(profile_id,outbox_id);

INSERT INTO ingest_events SELECT * FROM old_ingest_events;
INSERT INTO ingest_event_details SELECT * FROM old_ingest_event_details;
INSERT INTO ingest_observations SELECT * FROM old_ingest_observations;
INSERT INTO ingest_revisions SELECT * FROM old_ingest_revisions;
INSERT INTO outbox SELECT * FROM old_outbox;
INSERT INTO outbox_details SELECT * FROM old_outbox_details;

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_inactive BEFORE INSERT ON ingest_events
WHEN NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
  OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
  OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
  OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
BEGIN
    SELECT RAISE(ABORT,'Receipt resolution requires pipeline activation');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_inactive_update BEFORE UPDATE ON ingest_events
WHEN NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
  OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
  OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
  OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
BEGIN
    SELECT RAISE(ABORT,'Receipt resolution requires pipeline activation');
END;
-- +goose StatementEnd

-- The legacy writers still operate on tracks and use the full listen conflict index.
-- +goose StatementBegin
CREATE TRIGGER trg_identity_track_insert AFTER INSERT ON tracks
BEGIN
    SELECT RAISE(ABORT,'Occurrence ID already exists')
      WHERE EXISTS (SELECT 1 FROM occurrences WHERE id=NEW.id);
    INSERT INTO recordings (musicbrainz_id) VALUES (NEW.musicbrainz_id);
    INSERT INTO occurrences (id,recording_id,release_id,legacy_track_id)
      VALUES (NEW.id,last_insert_rowid(),NEW.release_id,NEW.id);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_track_update AFTER UPDATE OF release_id,musicbrainz_id ON tracks
BEGIN
    UPDATE occurrences SET release_id=NEW.release_id WHERE legacy_track_id=NEW.id;
    UPDATE recordings SET musicbrainz_id=NEW.musicbrainz_id, mbid_source='legacy_declared'
      WHERE id=(SELECT recording_id FROM occurrences WHERE legacy_track_id=NEW.id)
        AND NEW.musicbrainz_id IS NOT OLD.musicbrainz_id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_track_delete AFTER DELETE ON tracks
BEGIN
    DELETE FROM occurrences WHERE legacy_track_id=OLD.id;
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_recording_orphan AFTER DELETE ON occurrences
BEGIN
    DELETE FROM recordings WHERE id=OLD.recording_id
      AND NOT EXISTS (SELECT 1 FROM occurrences WHERE recording_id=OLD.recording_id);
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_legacy_occurrence BEFORE INSERT ON occurrences
WHEN NEW.release_id IS NULL OR NEW.legacy_track_id IS NULL OR NOT EXISTS (
    SELECT 1 FROM tracks WHERE id=NEW.legacy_track_id AND release_id=NEW.release_id)
BEGIN
    SELECT RAISE(ABORT,'Standalone occurrences require activation');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_legacy_listen_insert BEFORE INSERT ON listens
WHEN NEW.origin IS NOT 'legacy'
BEGIN
    SELECT RAISE(ABORT,'New listen origins require activation');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_legacy_listen_update BEFORE UPDATE OF origin ON listens
WHEN NEW.origin IS NOT 'legacy'
BEGIN
    SELECT RAISE(ABORT,'New listen origins require activation');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_track_id_immutable BEFORE UPDATE ON tracks
WHEN NEW.id IS NOT OLD.id
BEGIN
    SELECT RAISE(ABORT,'Legacy track IDs are immutable');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_identity_event_id_immutable BEFORE UPDATE ON listens
WHEN NEW.event_id IS NOT OLD.event_id
BEGIN
    SELECT RAISE(ABORT,'Event IDs are immutable');
END;
-- +goose StatementEnd

-- Assert both cardinality and exact preservation of the legacy rows.
INSERT INTO identity_assertion VALUES ('listens per profile',
    NOT EXISTS (SELECT profile_id,count(*) FROM listens GROUP BY profile_id
                EXCEPT SELECT profile_id,count(*) FROM old_listens GROUP BY profile_id)
    AND NOT EXISTS (SELECT profile_id,count(*) FROM old_listens GROUP BY profile_id
                    EXCEPT SELECT profile_id,count(*) FROM listens GROUP BY profile_id));
INSERT INTO identity_assertion VALUES ('listen contents',
    NOT EXISTS (SELECT rowid,profile_id,track_id,listened_at,user_id,client FROM listens
                EXCEPT SELECT * FROM old_listens)
    AND NOT EXISTS (SELECT * FROM old_listens
                    EXCEPT SELECT rowid,profile_id,track_id,listened_at,user_id,client FROM listens));
INSERT INTO identity_assertion VALUES ('foreign keys', NOT EXISTS (SELECT 1 FROM pragma_foreign_key_check));
INSERT INTO identity_assertion VALUES ('ingest_events',
    (SELECT count(*) FROM ingest_events)=(SELECT count(*) FROM old_ingest_events)
    AND NOT EXISTS (SELECT * FROM ingest_events EXCEPT SELECT * FROM old_ingest_events)
    AND NOT EXISTS (SELECT * FROM old_ingest_events EXCEPT SELECT * FROM ingest_events));
INSERT INTO identity_assertion VALUES ('ingest_event_details',
    (SELECT count(*) FROM ingest_event_details)=(SELECT count(*) FROM old_ingest_event_details)
    AND NOT EXISTS (SELECT * FROM ingest_event_details EXCEPT SELECT * FROM old_ingest_event_details)
    AND NOT EXISTS (SELECT * FROM old_ingest_event_details EXCEPT SELECT * FROM ingest_event_details));
INSERT INTO identity_assertion VALUES ('ingest_observations',
    (SELECT count(*) FROM ingest_observations)=(SELECT count(*) FROM old_ingest_observations)
    AND NOT EXISTS (SELECT * FROM ingest_observations EXCEPT SELECT * FROM old_ingest_observations)
    AND NOT EXISTS (SELECT * FROM old_ingest_observations EXCEPT SELECT * FROM ingest_observations));
INSERT INTO identity_assertion VALUES ('ingest_revisions',
    (SELECT count(*) FROM ingest_revisions)=(SELECT count(*) FROM old_ingest_revisions)
    AND NOT EXISTS (SELECT * FROM ingest_revisions EXCEPT SELECT * FROM old_ingest_revisions)
    AND NOT EXISTS (SELECT * FROM old_ingest_revisions EXCEPT SELECT * FROM ingest_revisions));
INSERT INTO identity_assertion VALUES ('outbox',
    (SELECT count(*) FROM outbox)=(SELECT count(*) FROM old_outbox)
    AND NOT EXISTS (SELECT * FROM outbox EXCEPT SELECT * FROM old_outbox)
    AND NOT EXISTS (SELECT * FROM old_outbox EXCEPT SELECT * FROM outbox));
INSERT INTO identity_assertion VALUES ('outbox_details',
    (SELECT count(*) FROM outbox_details)=(SELECT count(*) FROM old_outbox_details)
    AND NOT EXISTS (SELECT * FROM outbox_details EXCEPT SELECT * FROM old_outbox_details)
    AND NOT EXISTS (SELECT * FROM old_outbox_details EXCEPT SELECT * FROM outbox_details));
DROP TABLE old_ingest_events;
DROP TABLE old_ingest_event_details;
DROP TABLE old_ingest_observations;
DROP TABLE old_ingest_revisions;
DROP TABLE old_outbox;
DROP TABLE old_outbox_details;
DROP TABLE old_listens;
DROP TABLE old_track_map;
DROP TABLE identity_assertion;

-- +goose Down
-- Restoring a verified backup requires explicit reconciliation.
CREATE TEMP TABLE identity_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO identity_downgrade_guard VALUES (1);
