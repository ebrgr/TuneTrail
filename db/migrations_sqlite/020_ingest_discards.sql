-- +goose Up
-- Revisions cascade with the receipt. Keep only discard metadata under the
-- profile so the receipt and its payload can be deleted without losing the audit.
CREATE TABLE ingest_discards (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    receipt_id TEXT NOT NULL,
    actor_id INTEGER NOT NULL CHECK(actor_id > 0),
    discarded_at INTEGER NOT NULL,
    origin_state TEXT NOT NULL,
    park_reason TEXT,
    reason_codes TEXT NOT NULL CHECK(json_valid(reason_codes) AND json_type(reason_codes)='array'),
    PRIMARY KEY(profile_id,receipt_id)
) WITHOUT ROWID;

CREATE INDEX idx_ingest_discard_queue ON ingest_events(profile_id,received_at,receipt_id)
WHERE event_id IS NULL AND state IN ('received','retryable','pending_identity','pending_duplicate','invalid');

-- +goose Down
CREATE TEMP TABLE ingest_discards_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO ingest_discards_downgrade_guard VALUES (1);
