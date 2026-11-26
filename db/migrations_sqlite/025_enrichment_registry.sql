-- +goose Up
-- ISSUE-214: enrichment sources become rows in a registry instead of code paths. The row chooses
-- whether a provider runs and where in the order; the process decides whether it is available;
-- the Go descriptor decides what it can do. Adding or removing a source needs no migration: the
-- engine inserts missing rows at startup. Migration 024 never reached a production database, and
-- the single consolidated migration (ISSUE-195) folds this one in.
CREATE TABLE enrichment_providers (
    id TEXT PRIMARY KEY CHECK(length(id)>0),
    enabled INTEGER NOT NULL DEFAULT 1 CHECK(enabled IN (0,1)),
    available INTEGER NOT NULL DEFAULT 0 CHECK(available IN (0,1)),
    priority INTEGER NOT NULL CHECK(priority>=0),
    identity INTEGER NOT NULL CHECK(identity IN (0,1)),
    metadata INTEGER NOT NULL CHECK(metadata IN (0,1)),
    eligibility TEXT NOT NULL CHECK(eligibility IN ('configured','profile-relay')),
    CHECK(identity+metadata>0)
) WITHOUT ROWID;

-- The old global switch meant "a Subsonic server is configured"; it becomes that provider's
-- availability. ListenBrainz was always attached, and each receipt still needs an approved relay.
INSERT INTO enrichment_providers(id,enabled,available,priority,identity,metadata,eligibility)
SELECT 'navidrome-subsonic',1,enabled,100,1,0,'configured' FROM enrichment_configuration WHERE singleton=1;
INSERT INTO enrichment_providers(id,enabled,available,priority,identity,metadata,eligibility)
VALUES ('listenbrainz',1,1,200,1,0,'profile-relay');

DROP TRIGGER trg_enrichment_enqueue;
DROP TABLE enrichment_configuration;

-- Only completed new API receipts enter this queue; historical scanning is ISSUE-213. A receipt is
-- queued when any provider that needs only the process configuration is enabled and available.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_enqueue AFTER UPDATE OF state ON ingest_events
WHEN NEW.state IN ('accepted','duplicate') AND OLD.state NOT IN ('accepted','duplicate')
     AND NEW.mode='active' AND NEW.source_type='listenbrainz'
     AND EXISTS (SELECT 1 FROM enrichment_providers WHERE enabled=1 AND available=1 AND eligibility='configured')
BEGIN
    INSERT OR IGNORE INTO enrichment_jobs(receipt_id) VALUES(NEW.receipt_id);
END;
-- +goose StatementEnd

-- One row per receipt, provider and chain records what each source answered. The job row keeps the
-- lease, the attempts and the combined state; a recovered job skips the providers that already
-- gave a final answer instead of repeating their network calls.
CREATE TABLE enrichment_provider_runs (
    receipt_id TEXT NOT NULL REFERENCES enrichment_jobs(receipt_id) ON DELETE CASCADE,
    provider TEXT NOT NULL REFERENCES enrichment_providers(id),
    chain TEXT NOT NULL CHECK(chain IN ('identity','metadata')),
    state TEXT NOT NULL CHECK(state IN ('found','absent','ambiguous','disabled','failed')),
    instance TEXT,
    finished_at INTEGER NOT NULL,
    PRIMARY KEY(receipt_id,provider,chain)
) WITHOUT ROWID;

-- +goose Down
CREATE TEMP TABLE enrichment_registry_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO enrichment_registry_downgrade_guard VALUES (1);
