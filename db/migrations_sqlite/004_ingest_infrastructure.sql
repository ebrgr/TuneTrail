-- +goose Up
-- Koito SQLite databases created before upstream PR #257 (2026-04-30) reached v2
-- without these indexes from 001. Recreate them so a Koito v2 database converts
-- straight to v4 and passes the layout validation (ISSUE-080).
CREATE INDEX IF NOT EXISTS idx_artist_aliases_artist_id ON artist_aliases(artist_id);
CREATE INDEX IF NOT EXISTS idx_listens_track_id_listened_at ON listens(track_id, listened_at);
-- Infrastructure only: identity writers remain disabled until ISSUE-133.
CREATE TABLE integrations (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)>0),
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    source_type TEXT NOT NULL CHECK(length(source_type)>0),
    display_name TEXT NOT NULL DEFAULT '',
    token_ref TEXT,
    revoked_at INTEGER,
    created_at INTEGER NOT NULL,
    details_retention_days INTEGER NOT NULL DEFAULT 90 CHECK(details_retention_days BETWEEN 7 AND 3650),
    diagnostic_enabled INTEGER NOT NULL DEFAULT 0 CHECK(diagnostic_enabled IN (0,1)),
    UNIQUE(profile_id,id)
);

-- The core and current structured snapshots do not expire with operational details.
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
    state TEXT NOT NULL DEFAULT 'received' CHECK(state IN ('received','pending_identity','pending_duplicate','invalid')),
    identity_status TEXT NOT NULL DEFAULT 'unresolved' CHECK(identity_status IN ('unresolved','pending','invalid')),
    duplicate_decision TEXT CHECK(duplicate_decision IN ('same','different','review')),
    reason_codes TEXT NOT NULL DEFAULT '[]' CHECK(json_valid(reason_codes) AND json_type(reason_codes)='array'),
    evidence TEXT NOT NULL DEFAULT '{}' CHECK(json_valid(evidence) AND json_type(evidence)='object'),
    rule_version TEXT,
    resolution_version INTEGER NOT NULL DEFAULT 0 CHECK(resolution_version=0),
    resolved_at INTEGER,
    event_id TEXT CHECK(event_id IS NULL),
    occurrence_id INTEGER CHECK(occurrence_id IS NULL),
    recording_id INTEGER CHECK(recording_id IS NULL),
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count>=0),
    next_attempt_at INTEGER,
    lease_owner TEXT,
    lease_until INTEGER,
    work_version INTEGER NOT NULL DEFAULT 0 CHECK(work_version>=0),
    closed_at INTEGER,
    UNIQUE(profile_id,receipt_id),
    FOREIGN KEY(profile_id,source_instance_id) REFERENCES integrations(profile_id,id) ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED,
    CHECK(external_id_trusted=0 OR (external_namespace IS NOT NULL AND external_event_id IS NOT NULL)),
    CHECK((external_namespace IS NULL)=(external_event_id IS NULL)),
    CHECK(client_idempotency_key IS NULL OR (batch_digest IS NOT NULL AND length(batch_digest)>0)),
    CHECK((file_digest IS NULL AND file_format IS NULL AND format_version IS NULL) OR
          (file_digest IS NOT NULL AND file_format IS NOT NULL AND length(file_format)>0 AND format_version IS NOT NULL AND length(format_version)>0)),
    CHECK((normalized_snapshot IS NULL)=(normalizer_version IS NULL)),
    CHECK((lease_owner IS NULL)=(lease_until IS NULL)),
    CHECK(closed_at IS NULL OR state='invalid'),
    CHECK(state!='invalid' OR (rule_version IS NOT NULL AND json_array_length(reason_codes)>0))
);
CREATE UNIQUE INDEX idx_ingest_external_key ON ingest_events(profile_id,source_instance_id,external_namespace,external_event_id) WHERE external_id_trusted=1;
CREATE UNIQUE INDEX idx_ingest_client_key ON ingest_events(profile_id,source_instance_id,operation,client_idempotency_key,item_ordinal) WHERE client_idempotency_key IS NOT NULL;
CREATE UNIQUE INDEX idx_ingest_file_key ON ingest_events(profile_id,source_instance_id,file_format,format_version,file_digest,item_ordinal) WHERE file_digest IS NOT NULL;
CREATE INDEX idx_ingest_queue ON ingest_events(state,next_attempt_at,received_at) WHERE closed_at IS NULL;
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
    SELECT RAISE(ABORT,'Receipt resolution requires ISSUE-133 identity migration');
END;
-- +goose StatementEnd

CREATE TABLE outbox (
    id TEXT PRIMARY KEY NOT NULL CHECK(length(id)>0),
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    destination_id TEXT NOT NULL,
    event_id TEXT NOT NULL CHECK(length(event_id)>0), -- Future opaque event identity, never track/second.
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
    SELECT RAISE(ABORT,'Outbox requires ISSUE-133 identity migration');
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

-- +goose Down
-- Restoring a verified backup is explicit; never silently discard future work.
CREATE TEMP TABLE ingest_downgrade_guard (value INTEGER);
-- +goose StatementBegin
CREATE TEMP TRIGGER ingest_downgrade_block BEFORE INSERT ON ingest_downgrade_guard
BEGIN
    SELECT RAISE(ABORT,'Restore the pre-ingest backup instead of downgrading');
END;
-- +goose StatementEnd
INSERT INTO ingest_downgrade_guard VALUES(1);
