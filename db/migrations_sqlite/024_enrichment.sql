-- +goose Up
-- Enrichment has its own terminal work state and never changes the receipt snapshot.
CREATE TABLE enrichment_configuration (
    singleton INTEGER PRIMARY KEY CHECK(singleton=1),
    enabled INTEGER NOT NULL CHECK(enabled IN (0,1))
);
INSERT INTO enrichment_configuration VALUES(1,0);
CREATE TABLE enrichment_jobs (
    receipt_id TEXT PRIMARY KEY REFERENCES ingest_events(receipt_id) ON DELETE CASCADE,
    state TEXT NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','working','found','absent','ambiguous','disabled','failed')),
    lease TEXT,
    lease_until INTEGER,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK(attempts>=0),
    instance TEXT,
    CHECK((lease IS NULL)=(lease_until IS NULL))
);
CREATE INDEX idx_enrichment_pending ON enrichment_jobs(state,lease_until);

-- Claims stay attached to the profile and actual catalog entity, not the shared MBID column.
CREATE TABLE enrichment_recording_claims (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    receipt_id TEXT NOT NULL REFERENCES ingest_events(receipt_id) ON DELETE CASCADE,
    instance TEXT NOT NULL,
    mbid TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY(profile_id,occurrence_id,receipt_id,instance,mbid)
);
CREATE TABLE enrichment_artist_claims (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER NOT NULL REFERENCES artists(id) ON DELETE CASCADE,
    receipt_id TEXT NOT NULL REFERENCES ingest_events(receipt_id) ON DELETE CASCADE,
    instance TEXT NOT NULL,
    mbid TEXT NOT NULL,
    source TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    PRIMARY KEY(profile_id,artist_id,receipt_id,instance,mbid)
);

-- Only completed new API receipts enter this queue; historical scanning is ISSUE-213.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_enqueue AFTER UPDATE OF state ON ingest_events
WHEN NEW.state IN ('accepted','duplicate') AND OLD.state NOT IN ('accepted','duplicate')
     AND NEW.mode='active' AND NEW.source_type='listenbrainz'
     AND (SELECT enabled FROM enrichment_configuration WHERE singleton=1)=1
BEGIN
    INSERT OR IGNORE INTO enrichment_jobs(receipt_id) VALUES(NEW.receipt_id);
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE enrichment_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO enrichment_downgrade_guard VALUES (1);
