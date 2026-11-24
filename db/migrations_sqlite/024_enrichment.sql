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

-- Project only the credit names that actually carry received IDs.
CREATE TABLE enrichment_received_artist_credits (
    receipt_id TEXT NOT NULL REFERENCES ingest_events(receipt_id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    profile_id INTEGER NOT NULL,
    occurrence_id INTEGER NOT NULL,
    PRIMARY KEY(receipt_id,name)
) WITHOUT ROWID;
CREATE INDEX idx_enrichment_received_credit
    ON enrichment_received_artist_credits(profile_id,occurrence_id,name);

INSERT INTO enrichment_received_artist_credits
SELECT e.receipt_id,n.value,e.profile_id,e.occurrence_id
FROM ingest_events e CROSS JOIN json_each(tt_received_artist_names(e.original_snapshot)) n
WHERE e.state IN ('accepted','duplicate') AND e.occurrence_id IS NOT NULL;

-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_received_credit_insert AFTER INSERT ON ingest_events
WHEN NEW.state IN ('accepted','duplicate') AND NEW.occurrence_id IS NOT NULL
BEGIN
    INSERT INTO enrichment_received_artist_credits
    SELECT NEW.receipt_id,value,NEW.profile_id,NEW.occurrence_id
    FROM json_each(tt_received_artist_names(NEW.original_snapshot));
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_received_credit_update AFTER UPDATE OF state,occurrence_id ON ingest_events
WHEN NEW.occurrence_id IS NOT OLD.occurrence_id
  OR (NEW.state IN ('accepted','duplicate')) <> (OLD.state IN ('accepted','duplicate'))
BEGIN
    DELETE FROM enrichment_received_artist_credits WHERE receipt_id=OLD.receipt_id;
    INSERT INTO enrichment_received_artist_credits
    SELECT NEW.receipt_id,value,NEW.profile_id,NEW.occurrence_id
    FROM json_each(tt_received_artist_names(NEW.original_snapshot))
    WHERE NEW.state IN ('accepted','duplicate') AND NEW.occurrence_id IS NOT NULL;
END;
-- +goose StatementEnd

CREATE INDEX idx_listens_profile_track ON listens(profile_id,track_id);

CREATE INDEX idx_ingest_received_recording ON ingest_events(profile_id,recording_id)
    WHERE state IN ('accepted','duplicate') AND
    (coalesce(json_extract(original_snapshot,'$.recording_mbid'),'')<>'' OR coalesce(json_extract(original_snapshot,'$.mapped_recording_mbid'),'')<>'');


CREATE TABLE metadata_values (
    id INTEGER PRIMARY KEY,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    recording_id INTEGER REFERENCES recordings(id) ON DELETE CASCADE,
    release_id INTEGER REFERENCES releases(id) ON DELETE CASCADE,
    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES listens(event_id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('mbid','genre','tag','recording_year','release_date','artist_origin_country')),
    value TEXT NOT NULL CHECK(length(value)>0),
    instance TEXT NOT NULL CHECK(length(instance)>0),
    source TEXT NOT NULL CHECK(length(source)>0),
    observed_at TEXT NOT NULL CHECK(length(observed_at)>0),
    CHECK((artist_id IS NOT NULL)+(recording_id IS NOT NULL)+(release_id IS NOT NULL)=1),
    CHECK((recording_id IS NULL)=(occurrence_id IS NULL)),
    CHECK(kind NOT IN ('recording_year','release_date','artist_origin_country')
        OR (kind='recording_year' AND recording_id IS NOT NULL)
        OR (kind='release_date' AND release_id IS NOT NULL)
        OR (kind='artist_origin_country' AND artist_id IS NOT NULL))
);
CREATE UNIQUE INDEX idx_metadata_artist
    ON metadata_values(profile_id,artist_id,kind,instance,value,source,event_id) WHERE artist_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_recording
    ON metadata_values(profile_id,occurrence_id,kind,instance,value,source,event_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_read
    ON metadata_values(profile_id,recording_id,kind,instance,value) WHERE recording_id IS NOT NULL;
CREATE UNIQUE INDEX idx_metadata_release
    ON metadata_values(profile_id,release_id,kind,instance,value,source,event_id) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_artist_fk ON metadata_values(artist_id) WHERE artist_id IS NOT NULL;
CREATE INDEX idx_metadata_recording_fk ON metadata_values(recording_id) WHERE recording_id IS NOT NULL;
CREATE INDEX idx_metadata_release_fk ON metadata_values(release_id) WHERE release_id IS NOT NULL;
CREATE INDEX idx_metadata_occurrence_fk ON metadata_values(occurrence_id) WHERE occurrence_id IS NOT NULL;
CREATE INDEX idx_metadata_event_fk ON metadata_values(event_id);

-- Defensive guards also apply to future providers and archive restores.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_scope_insert BEFORE INSERT ON metadata_values
BEGIN
    SELECT CASE WHEN NEW.recording_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM occurrences WHERE id=NEW.occurrence_id AND recording_id=NEW.recording_id
    ) THEN RAISE(ABORT,'Metadata recording does not match occurrence') END;
    SELECT CASE WHEN NEW.event_id IS NOT NULL AND NOT EXISTS (
        SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id
        WHERE l.event_id=NEW.event_id AND l.profile_id=NEW.profile_id AND (
            NEW.occurrence_id=o.id OR NEW.release_id=o.release_id
            OR EXISTS (SELECT 1 FROM artist_tracks WHERE track_id=o.id AND artist_id=NEW.artist_id)
            OR EXISTS (SELECT 1 FROM occurrence_credits WHERE occurrence_id=o.id AND artist_id=NEW.artist_id))
    ) THEN RAISE(ABORT,'Metadata event does not support entity and profile') END;
END;
-- +goose StatementEnd

-- Only the recording reconciliation trigger changes an existing entity binding.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_scope_update BEFORE UPDATE ON metadata_values
WHEN NEW.profile_id IS NOT OLD.profile_id OR NEW.artist_id IS NOT OLD.artist_id
  OR NEW.release_id IS NOT OLD.release_id OR NEW.occurrence_id IS NOT OLD.occurrence_id
  OR NEW.event_id IS NOT OLD.event_id
  OR (NEW.recording_id IS NOT OLD.recording_id AND NOT EXISTS (
        SELECT 1 FROM occurrences WHERE id=NEW.occurrence_id AND recording_id=NEW.recording_id))
BEGIN
    SELECT RAISE(ABORT,'Metadata observation binding is immutable');
END;
-- +goose StatementEnd

-- Occurrences stay put when a recording merge is applied or undone.
-- +goose StatementBegin
CREATE TRIGGER trg_metadata_recording_reconcile AFTER UPDATE OF recording_id ON occurrences
WHEN NEW.recording_id IS NOT OLD.recording_id
BEGIN
    UPDATE metadata_values SET recording_id=NEW.recording_id WHERE occurrence_id=NEW.id;
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE metadata_origin_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO metadata_origin_downgrade_guard VALUES (1);
