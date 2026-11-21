-- +goose Up
ALTER TABLE ingest_events ADD COLUMN parked_at INTEGER;
ALTER TABLE ingest_events ADD COLUMN park_reason TEXT
    CHECK((parked_at IS NULL AND park_reason IS NULL) OR
          (parked_at IS NOT NULL AND park_reason IS NOT NULL AND mode='active'
           AND state='retryable' AND closed_at IS NULL AND lease_owner IS NULL AND lease_until IS NULL));
-- attempt_count remains the lifetime count. The base tracks grants of a fresh budget
-- and lease recoveries, which resume the interrupted attempt.
ALTER TABLE ingest_events ADD COLUMN retry_attempt_base INTEGER NOT NULL DEFAULT 0
    CHECK(retry_attempt_base BETWEEN 0 AND attempt_count);

CREATE INDEX idx_ingest_resolver_queue ON ingest_events(received_at,receipt_id)
    WHERE mode='active' AND closed_at IS NULL AND parked_at IS NULL AND state IN ('received','retryable');

CREATE INDEX idx_ingest_parked ON ingest_events(profile_id,parked_at DESC,receipt_id DESC)
    WHERE parked_at IS NOT NULL;

-- +goose Down
-- Restore the pre-upgrade backup to downgrade without losing parked work.
CREATE TEMP TABLE ingest_retry_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO ingest_retry_downgrade_guard VALUES (1);
