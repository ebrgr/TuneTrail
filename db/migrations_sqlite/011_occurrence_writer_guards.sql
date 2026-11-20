-- +goose Up
-- Protect the one-way cut even when a writer bypasses admission.
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_receipt_admission BEFORE INSERT ON ingest_events
WHEN (NEW.resolver_variant='occurrence' AND (NEW.mode<>'active' OR NOT EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)))
  OR (NEW.mode='active' AND NEW.resolver_variant='v9' AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1))
BEGIN
    SELECT RAISE(ABORT,'Receipt variant conflicts with occurrence writing');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_listen_admission BEFORE INSERT ON listens
WHEN NEW.origin='legacy' AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)
BEGIN
    SELECT RAISE(ABORT,'Legacy listen conflicts with occurrence writing');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_cut_insert BEFORE INSERT ON profile_ingest_flags
WHEN NEW.occurrence_write=1 AND (
    EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='active' AND resolver_variant='v9' AND closed_at IS NULL)
    OR EXISTS (SELECT 1 FROM profile_relays WHERE profile_id=NEW.profile_id AND (url<>'' OR token<>'')))
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing requires drained receipts and no relay');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_cut_update BEFORE UPDATE OF occurrence_write ON profile_ingest_flags
WHEN OLD.occurrence_write=0 AND NEW.occurrence_write=1 AND (
    EXISTS (SELECT 1 FROM ingest_events WHERE profile_id=NEW.profile_id AND mode='active' AND resolver_variant='v9' AND closed_at IS NULL)
    OR EXISTS (SELECT 1 FROM profile_relays WHERE profile_id=NEW.profile_id AND (url<>'' OR token<>'')))
BEGIN
    SELECT RAISE(ABORT,'Occurrence writing requires drained receipts and no relay');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_relay_insert BEFORE INSERT ON profile_relays
WHEN (NEW.url<>'' OR NEW.token<>'') AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)
BEGIN
    SELECT RAISE(ABORT,'Relay conflicts with occurrence writing');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_relay_update BEFORE UPDATE ON profile_relays
WHEN (NEW.url<>'' OR NEW.token<>'') AND EXISTS (
    SELECT 1 FROM profile_ingest_flags WHERE profile_id=NEW.profile_id AND occurrence_write=1)
BEGIN
    SELECT RAISE(ABORT,'Relay conflicts with occurrence writing');
END;
-- +goose StatementEnd

-- v9 can link only legacy events; the new writer must match its adapter's origin.
-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_link_origin_insert BEFORE INSERT ON ingest_events
WHEN NEW.mode='active' AND NEW.event_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM listens l WHERE (l.profile_id,l.event_id)=(NEW.profile_id,NEW.event_id)
      AND l.origin=CASE
        WHEN NEW.resolver_variant='v9' THEN 'legacy'
        WHEN NEW.source_type='listenbrainz' AND NEW.file_format IS NULL THEN 'listenbrainz_api'
        WHEN NEW.source_type='file' AND NEW.file_format='listenbrainz' THEN 'listenbrainz_import'
        WHEN NEW.source_type='file' AND NEW.file_format='koito' THEN 'native_import'
        WHEN NEW.source_type='file' AND NEW.file_format IN ('lastfm','spotify','maloja') THEN NEW.file_format||'_import'
      END)
BEGIN
    SELECT RAISE(ABORT,'Ingest link origin conflicts with its resolver variant');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_occurrence_link_origin_update BEFORE UPDATE OF event_id,state ON ingest_events
WHEN NEW.mode='active' AND NEW.event_id IS NOT NULL AND NOT EXISTS (
    SELECT 1 FROM listens l WHERE (l.profile_id,l.event_id)=(NEW.profile_id,NEW.event_id)
      AND l.origin=CASE
        WHEN NEW.resolver_variant='v9' THEN 'legacy'
        WHEN NEW.source_type='listenbrainz' AND NEW.file_format IS NULL THEN 'listenbrainz_api'
        WHEN NEW.source_type='file' AND NEW.file_format='listenbrainz' THEN 'listenbrainz_import'
        WHEN NEW.source_type='file' AND NEW.file_format='koito' THEN 'native_import'
        WHEN NEW.source_type='file' AND NEW.file_format IN ('lastfm','spotify','maloja') THEN NEW.file_format||'_import'
      END)
BEGIN
    SELECT RAISE(ABORT,'Ingest link origin conflicts with its resolver variant');
END;
-- +goose StatementEnd

-- +goose Down
-- Restore the pre-upgrade backup instead of removing the writer boundary.
CREATE TEMP TABLE occurrence_writer_downgrade_guard (value INTEGER CHECK(value=0));
INSERT INTO occurrence_writer_downgrade_guard VALUES (1);
