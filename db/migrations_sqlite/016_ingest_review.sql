-- +goose Up
-- Keep automatic legacy deduplication while allowing explicit separate listens.
ALTER TABLE listens ADD COLUMN review_distinct INTEGER NOT NULL DEFAULT 0 CHECK(review_distinct IN (0,1));
DROP INDEX idx_listens_legacy_identity;
CREATE UNIQUE INDEX idx_listens_legacy_identity ON listens(profile_id,track_id,listened_at)
    WHERE origin='legacy' AND review_distinct=0;

CREATE INDEX idx_ingest_health_received ON ingest_events(profile_id,received_at DESC,receipt_id DESC);
CREATE INDEX idx_ingest_health_decisions ON ingest_events(profile_id,coalesce(resolved_at,received_at) DESC,receipt_id DESC)
    WHERE duplicate_decision IS NOT NULL;
CREATE INDEX idx_ingest_health_review ON ingest_events(profile_id,received_at DESC,receipt_id DESC)
    WHERE mode='active' AND state='pending_duplicate' AND closed_at IS NULL;

-- +goose Down
-- Older writers cannot represent separately reviewed legacy listens.
CREATE TEMP TABLE ingest_review_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO ingest_review_downgrade_guard VALUES (1);
