-- +goose Up
ALTER TABLE ingest_events ADD COLUMN claim_attempt_base INTEGER NOT NULL DEFAULT 0
    CHECK(claim_attempt_base BETWEEN 0 AND attempt_count);
ALTER TABLE ingest_events ADD COLUMN reprocessed_at INTEGER;
ALTER TABLE ingest_events ADD COLUMN reprocessed_by INTEGER
    CHECK((reprocessed_at IS NULL AND reprocessed_by IS NULL) OR
          (reprocessed_at IS NOT NULL AND reprocessed_by IS NOT NULL AND reprocessed_by>0));

-- The old retry base mixes manual resets with lease recoveries. Start a new claim budget
-- for existing receipts while preserving their lifetime claim count and retry state.
UPDATE ingest_events SET claim_attempt_base=attempt_count
    WHERE attempt_count>0 AND mode='active' AND closed_at IS NULL AND state IN ('received','retryable');

-- +goose Down
CREATE TEMP TABLE ingest_reprocess_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO ingest_reprocess_downgrade_guard VALUES (1);
