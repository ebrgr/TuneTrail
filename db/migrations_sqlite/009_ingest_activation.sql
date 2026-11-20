-- +goose Up
-- Existing receipts remain shadow; mode is fixed at admission.
ALTER TABLE ingest_events ADD COLUMN mode TEXT NOT NULL DEFAULT 'shadow'
    CHECK(mode IN ('shadow','active'));

CREATE INDEX idx_ingest_active_queue ON ingest_events(state,next_attempt_at,received_at)
    WHERE mode='active' AND closed_at IS NULL;

CREATE TABLE profile_ingest_flags (
    profile_id INTEGER PRIMARY KEY NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    active INTEGER NOT NULL DEFAULT 0 CHECK(active IN (0,1)),
    review INTEGER NOT NULL DEFAULT 0 CHECK(review IN (0,1))
);

DROP TRIGGER trg_ingest_event_inactive;
DROP TRIGGER trg_ingest_event_inactive_update;
DROP TRIGGER trg_ingest_revision_inactive;
DROP TRIGGER trg_outbox_inactive;

-- Keep the legacy shadow transitions and prevent promotion by changing a receipt's mode.
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_shadow_insert BEFORE INSERT ON ingest_events
WHEN NEW.mode='shadow' AND (
    NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
    OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
    OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
    OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
)
BEGIN
    SELECT RAISE(ABORT,'Shadow receipt cannot materialize an ingest decision');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_event_mode_update BEFORE UPDATE ON ingest_events
WHEN NEW.mode IS NOT OLD.mode OR (NEW.mode='shadow' AND (
    NEW.state NOT IN ('received','pending_identity','pending_duplicate','invalid')
    OR NEW.identity_status NOT IN ('unresolved','pending','invalid')
    OR NEW.event_id IS NOT NULL OR NEW.occurrence_id IS NOT NULL OR NEW.recording_id IS NOT NULL
    OR NEW.resolution_version<>0 OR (NEW.closed_at IS NOT NULL AND NEW.state<>'invalid')
))
BEGIN
    SELECT RAISE(ABORT,'Receipt mode is immutable and shadow receipts cannot materialize decisions');
END;
-- +goose StatementEnd

-- Decisions already admitted as active can finish after the profile opts out.
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_revision_active_insert BEFORE INSERT ON ingest_revisions
WHEN NOT EXISTS (
    SELECT 1 FROM ingest_events
    WHERE profile_id=NEW.profile_id AND receipt_id=NEW.receipt_id AND mode='active'
)
BEGIN
    SELECT RAISE(ABORT,'Receipt revision requires an active receipt');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_outbox_active_insert BEFORE INSERT ON outbox
WHEN NOT EXISTS (
    SELECT 1 FROM ingest_events
    WHERE profile_id=NEW.profile_id AND event_id=NEW.event_id AND mode='active' AND state='accepted'
)
BEGIN
    SELECT RAISE(ABORT,'Outbox intent requires an accepted active receipt');
END;
-- +goose StatementEnd

-- +goose Down
-- Reverting the schema requires restoring the pre-activation backup.
CREATE TEMP TABLE ingest_activation_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO ingest_activation_downgrade_guard VALUES (1);
