-- +goose Up
-- Refuse to discard development receipts that still need an explicit decision.
CREATE TABLE occurrence_upgrade_guard (id INTEGER PRIMARY KEY);
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_upgrade_guard BEFORE INSERT ON occurrence_upgrade_guard
WHEN EXISTS (
    SELECT 1 FROM ingest_events
    WHERE (mode='shadow' AND event_id IS NULL)
       OR (mode='active' AND resolver_variant='v9' AND closed_at IS NULL)
)
BEGIN
    SELECT RAISE(ABORT,'Schema 021 requires resolving open legacy receipts and clearing unlinked shadow receipts with the previous release. Back up the database first.');
END;
-- +goose StatementEnd
INSERT INTO occurrence_upgrade_guard VALUES (1);
DROP TABLE occurrence_upgrade_guard;

DROP TRIGGER trg_occurrence_receipt_admission;
DROP TRIGGER trg_occurrence_listen_admission;
DROP TRIGGER trg_occurrence_relay_insert;
DROP TRIGGER trg_occurrence_relay_update;
DROP TABLE profile_ingest_flags;
DROP INDEX idx_listens_legacy_identity;

-- Missing overrides mean enabled, including for profiles created after the migration.
CREATE TABLE profile_ingest_review (
    profile_id INTEGER PRIMARY KEY REFERENCES listening_profiles(id) ON DELETE CASCADE,
    review INTEGER NOT NULL DEFAULT 1 CHECK (review IN (0,1))
);

INSERT INTO event_origins VALUES ('manual','Manual listen') ON CONFLICT(origin) DO NOTHING;

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_receipt_admission BEFORE INSERT ON ingest_events
WHEN NEW.resolver_variant<>'occurrence' OR NEW.mode<>'active'
BEGIN
    SELECT RAISE(ABORT,'New receipts require active occurrence ingestion');
END;
-- +goose StatementEnd

-- A draft never becomes eligible by timing alone or by a receipt resolving later.
ALTER TABLE relay_destinations ADD COLUMN awaiting_approval INTEGER NOT NULL DEFAULT 0 CHECK(awaiting_approval IN (0,1));
ALTER TABLE ingest_events ADD COLUMN relay_admission_generation INTEGER;
-- Keep already-configured destinations and their pre-upgrade receipts.
UPDATE ingest_events SET relay_admission_generation=(
    SELECT generation FROM relay_destinations d JOIN profile_relays r USING(profile_id)
    WHERE d.profile_id=ingest_events.profile_id AND r.url<>'' AND r.token<>'')
WHERE source_type='listenbrainz' AND mode='active' AND closed_at IS NULL;

-- +goose StatementBegin
CREATE TRIGGER trg_relay_draft_insert AFTER INSERT ON profile_relays
BEGIN
    INSERT INTO relay_destinations(profile_id,awaiting_approval) VALUES(NEW.profile_id,1)
    ON CONFLICT(profile_id) DO UPDATE SET awaiting_approval=1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_draft_change AFTER UPDATE ON profile_relays
WHEN tt_relay_canonical(OLD.url) IS NOT tt_relay_canonical(NEW.url) OR OLD.token IS NOT NEW.token
BEGIN
    UPDATE relay_destinations SET awaiting_approval=1,
        generation=generation+(tt_relay_canonical(OLD.url) IS tt_relay_canonical(NEW.url)),
        open_count=0 WHERE profile_id=NEW.profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_relay_admission_generation AFTER INSERT ON ingest_events
WHEN NEW.mode='active' AND NEW.source_type='listenbrainz' AND NEW.file_format IS NULL
BEGIN
    UPDATE ingest_events SET relay_admission_generation=(
        SELECT generation FROM relay_destinations d JOIN profile_relays r USING(profile_id)
        WHERE d.profile_id=NEW.profile_id AND d.awaiting_approval=0 AND r.url<>'' AND r.token<>'')
    WHERE receipt_id=NEW.receipt_id;
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE occurrence_only_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO occurrence_only_downgrade_guard VALUES (1);
