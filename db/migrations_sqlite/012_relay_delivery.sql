-- +goose Up
CREATE TABLE relay_delivery_gate (
    id INTEGER PRIMARY KEY CHECK(id=1),
    policy TEXT NOT NULL CHECK(policy IN ('legacy','durable'))
);
INSERT INTO relay_delivery_gate VALUES(1,'legacy');

ALTER TABLE ingest_events ADD COLUMN delivery_policy TEXT NOT NULL DEFAULT 'legacy'
    CHECK(delivery_policy IN ('legacy','durable'));
ALTER TABLE outbox ADD COLUMN format_version INTEGER NOT NULL DEFAULT 0 CHECK(format_version>=0);
ALTER TABLE outbox ADD COLUMN last_attempt_at INTEGER;
ALTER TABLE outbox ADD COLUMN relay_generation INTEGER NOT NULL DEFAULT 1 CHECK(relay_generation>0);

CREATE TABLE relay_destinations (
    profile_id INTEGER PRIMARY KEY REFERENCES listening_profiles(id) ON DELETE CASCADE,
    generation INTEGER NOT NULL DEFAULT 1,
    config_version INTEGER NOT NULL DEFAULT 1,
    open_count INTEGER NOT NULL DEFAULT 0 CHECK(open_count BETWEEN 0 AND 100000),
    paused_until INTEGER,
    pause_class TEXT,
    backoff_count INTEGER NOT NULL DEFAULT 0 CHECK(backoff_count>=0),
    lease_owner TEXT,
    lease_until INTEGER,
    work_version INTEGER NOT NULL DEFAULT 0,
    last_claim_at INTEGER NOT NULL DEFAULT 0,
    CHECK((lease_owner IS NULL)=(lease_until IS NULL))
);

-- Keep only daily aggregate counts after the underlying events are deleted.
CREATE TABLE relay_sent_archive (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    sent_day INTEGER NOT NULL,
    sent_count INTEGER NOT NULL CHECK(sent_count>0),
    PRIMARY KEY(profile_id,sent_day)
);

-- +goose StatementBegin
CREATE TRIGGER trg_relay_archive_sent BEFORE DELETE ON outbox
WHEN OLD.state='sent' AND OLD.format_version>0 AND OLD.destination_id='relay:'||OLD.profile_id
 AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=OLD.profile_id)
BEGIN
    INSERT INTO relay_sent_archive VALUES(OLD.profile_id,OLD.closed_at/86400,1)
    ON CONFLICT(profile_id,sent_day) DO UPDATE SET sent_count=sent_count+1;
END;
-- +goose StatementEnd

CREATE TABLE relay_allowlist (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    host_port TEXT NOT NULL,
    cidr TEXT NOT NULL DEFAULT '',
    allow_http INTEGER NOT NULL CHECK(allow_http IN (0,1)),
    allow_private INTEGER NOT NULL CHECK(allow_private IN (0,1)),
    origin TEXT NOT NULL CHECK(origin IN ('manual','seeded')),
    CHECK(allow_http=1 OR (allow_private=1 AND cidr<>'')),
    CHECK((allow_private=1)=(cidr<>'')),
    PRIMARY KEY(profile_id,host_port,cidr,origin)
);

INSERT INTO relay_destinations(profile_id, pause_class)
SELECT profile_id, 'policy_blocked' FROM profile_relays;

CREATE INDEX idx_outbox_relay_due ON outbox(profile_id,destination_id,relay_generation,next_attempt_at,id)
WHERE format_version>0 AND state IN ('pending','retryable');
CREATE INDEX idx_outbox_relay_capacity ON outbox(profile_id,destination_id,relay_generation,created_at,id)
WHERE format_version>0 AND state='failed' AND json_extract(result,'$.reason')='queue_capacity';

DROP TRIGGER trg_ingest_event_shadow_insert;
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_shadow_insert BEFORE INSERT ON ingest_events
WHEN NEW.mode='shadow' AND (
    NEW.delivery_policy<>'legacy' OR NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
    OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
    OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
    OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
)
BEGIN
    SELECT RAISE(ABORT,'Shadow receipt cannot materialize an ingest decision');
END;
-- +goose StatementEnd

DROP TRIGGER trg_ingest_event_mode_update;
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_mode_update BEFORE UPDATE ON ingest_events
WHEN NEW.delivery_policy IS NOT OLD.delivery_policy OR NEW.mode IS NOT OLD.mode OR (NEW.mode='shadow' AND (
    NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
    OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
    OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
    OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
))
BEGIN
    SELECT RAISE(ABORT,'Receipt mode is immutable and shadow receipts cannot materialize decisions');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_active_policy BEFORE INSERT ON ingest_events
WHEN NEW.mode='active' AND NEW.delivery_policy IS NOT (SELECT policy FROM relay_delivery_gate WHERE id=1)
BEGIN
    SELECT RAISE(ABORT,'Receipt delivery policy conflicts with the active writer');
END;
-- +goose StatementEnd

DROP TRIGGER trg_occurrence_cut_insert;
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_cut_insert BEFORE INSERT ON profile_ingest_flags
WHEN NEW.occurrence_write=1 AND (
    EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='active' AND resolver_variant='v9' AND closed_at IS NULL)
    OR EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='shadow' AND event_id IS NULL)
    OR ((SELECT policy FROM relay_delivery_gate WHERE id=1) IS NOT 'durable'
        AND EXISTS (SELECT 1 FROM profile_relays WHERE profile_id=NEW.profile_id AND (url<>'' OR token<>''))))
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing requires drained receipts, cleared shadow and no relay');
END;
-- +goose StatementEnd

DROP TRIGGER trg_occurrence_cut_update;
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_cut_update BEFORE UPDATE OF occurrence_write ON profile_ingest_flags
WHEN OLD.occurrence_write=0 AND NEW.occurrence_write=1 AND (
    EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='active' AND resolver_variant='v9' AND closed_at IS NULL)
    OR EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='shadow' AND event_id IS NULL)
    OR ((SELECT policy FROM relay_delivery_gate WHERE id=1) IS NOT 'durable'
        AND EXISTS (SELECT 1 FROM profile_relays WHERE profile_id=NEW.profile_id AND (url<>'' OR token<>''))))
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing requires drained receipts, cleared shadow and no relay');
END;
-- +goose StatementEnd

DROP TRIGGER trg_occurrence_relay_insert;
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_relay_insert BEFORE INSERT ON profile_relays
WHEN (SELECT policy FROM relay_delivery_gate WHERE id=1) IS NOT 'durable'
  AND (NEW.url<>'' OR NEW.token<>'') AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)
BEGIN
    SELECT RAISE(ABORT,'Relay conflicts with occurrence writing');
END;
-- +goose StatementEnd

DROP TRIGGER trg_occurrence_relay_update;
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_relay_update BEFORE UPDATE ON profile_relays
WHEN (SELECT policy FROM relay_delivery_gate WHERE id=1) IS NOT 'durable'
  AND (NEW.url<>'' OR NEW.token<>'') AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)
BEGIN
    SELECT RAISE(ABORT,'Relay conflicts with occurrence writing');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_destination_insert AFTER INSERT ON profile_relays
BEGIN
    INSERT INTO relay_destinations(profile_id) VALUES(NEW.profile_id)
    ON CONFLICT(profile_id) DO UPDATE SET config_version=config_version+1,
        pause_class=NULL, paused_until=NULL, backoff_count=0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_destination_update AFTER UPDATE ON profile_relays
WHEN tt_relay_canonical(OLD.url) IS NOT tt_relay_canonical(NEW.url) OR OLD.token IS NOT NEW.token
BEGIN
    UPDATE relay_destinations SET generation=generation+(tt_relay_canonical(OLD.url) IS NOT tt_relay_canonical(NEW.url)),
        open_count=CASE WHEN tt_relay_canonical(OLD.url) IS NOT tt_relay_canonical(NEW.url) THEN 0 ELSE open_count END,
        config_version=config_version+1, pause_class=NULL, paused_until=NULL, backoff_count=0
    WHERE profile_id=NEW.profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_destination_delete AFTER DELETE ON profile_relays
BEGIN
    UPDATE relay_destinations SET generation=generation+1,config_version=config_version+1,open_count=0,
        pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=OLD.profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_count_insert AFTER INSERT ON outbox
BEGIN
    UPDATE relay_destinations SET open_count=open_count+(NEW.format_version>0 AND NEW.state IN ('pending','retryable') AND NEW.destination_id='relay:'||NEW.profile_id AND NEW.relay_generation=generation)-0
      WHERE profile_id=NEW.profile_id AND (NEW.format_version>0 AND NEW.state IN ('pending','retryable') AND NEW.destination_id='relay:'||NEW.profile_id AND NEW.relay_generation=generation)<>0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_count_update AFTER UPDATE OF state,format_version,relay_generation ON outbox
BEGIN
    UPDATE relay_destinations SET open_count=open_count+(NEW.format_version>0 AND NEW.state IN ('pending','retryable') AND NEW.destination_id='relay:'||NEW.profile_id AND NEW.relay_generation=generation)-(OLD.format_version>0 AND OLD.state IN ('pending','retryable') AND OLD.destination_id='relay:'||OLD.profile_id AND OLD.relay_generation=generation)
      WHERE profile_id=NEW.profile_id AND (NEW.format_version>0 AND NEW.state IN ('pending','retryable') AND NEW.destination_id='relay:'||NEW.profile_id AND NEW.relay_generation=generation)<>(OLD.format_version>0 AND OLD.state IN ('pending','retryable') AND OLD.destination_id='relay:'||OLD.profile_id AND OLD.relay_generation=generation);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_count_delete AFTER DELETE ON outbox
BEGIN
    UPDATE relay_destinations SET open_count=open_count+0-(OLD.format_version>0 AND OLD.state IN ('pending','retryable') AND OLD.destination_id='relay:'||OLD.profile_id AND OLD.relay_generation=generation)
      WHERE profile_id=OLD.profile_id AND 0<>(OLD.format_version>0 AND OLD.state IN ('pending','retryable') AND OLD.destination_id='relay:'||OLD.profile_id AND OLD.relay_generation=generation);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_policy_one_way BEFORE UPDATE OF policy ON relay_delivery_gate
WHEN OLD.policy='durable' AND NEW.policy<>'durable'
BEGIN
    SELECT RAISE(ABORT,'Durable relay admission cannot be reversed');
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE relay_delivery_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO relay_delivery_downgrade_guard VALUES (1);
