-- +goose Up
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
DROP TRIGGER trg_enrichment_listen_delete;

-- A profile's declarations are removed when its last related listen is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_listen_delete AFTER DELETE ON listens
WHEN NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id)
BEGIN
    -- A recording can span occurrences; retain the declaration on remaining profile listens.
    INSERT OR IGNORE INTO enrichment_recording_claims(profile_id,occurrence_id,instance,mbid,source,observed_at)
    SELECT c.profile_id,other.id,c.instance,c.mbid,c.source,c.observed_at
    FROM enrichment_recording_claims c JOIN occurrences old_occurrence ON old_occurrence.id=c.occurrence_id
    JOIN occurrences other ON other.recording_id=old_occurrence.recording_id AND other.id<>old_occurrence.id
    WHERE c.profile_id=OLD.profile_id AND c.occurrence_id=OLD.track_id
      AND EXISTS (SELECT 1 FROM listens WHERE profile_id=c.profile_id AND track_id=other.id)
      AND EXISTS (SELECT 1 FROM listening_profiles WHERE id=OLD.profile_id);
    DELETE FROM enrichment_recording_claims WHERE profile_id=OLD.profile_id AND occurrence_id=OLD.track_id;
END;
-- +goose StatementEnd

-- Only the affected occurrence's artists can have lost a supporting listen.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_listen_delete AFTER DELETE ON listens
WHEN NOT EXISTS (SELECT 1 FROM listens WHERE profile_id=OLD.profile_id AND track_id=OLD.track_id)
BEGIN
    DELETE FROM enrichment_artist_claims
    WHERE profile_id=OLD.profile_id AND artist_id IN (
        SELECT artist_id FROM artist_tracks WHERE track_id=OLD.track_id
        UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.track_id AND artist_id IS NOT NULL
    )
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id
          AND l.profile_id=OLD.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id
          AND l.profile_id=OLD.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- Clean up before occurrence credits can disappear in the foreign-key cascade.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_occurrence_delete BEFORE DELETE ON occurrences
BEGIN
    DELETE FROM enrichment_artist_claims
    WHERE profile_id IN (SELECT profile_id FROM listens WHERE track_id=OLD.id)
    AND artist_id IN (
        SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id
        UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=OLD.id AND artist_id IS NOT NULL
    )
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id AND a.track_id<>OLD.id
          AND l.profile_id=enrichment_artist_claims.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id AND a.occurrence_id<>OLD.id
          AND l.profile_id=enrichment_artist_claims.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- A legacy track loses its artist relations before its occurrence is deleted.
-- +goose StatementBegin
CREATE TRIGGER trg_enrichment_artist_bridge_delete BEFORE DELETE ON tracks
BEGIN
    DELETE FROM enrichment_artist_claims
    WHERE profile_id IN (SELECT profile_id FROM listens WHERE track_id=OLD.id)
    AND artist_id IN (SELECT artist_id FROM artist_tracks WHERE track_id=OLD.id)
    AND NOT EXISTS (
        SELECT 1 FROM artist_tracks a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id AND a.track_id<>OLD.id
          AND l.profile_id=enrichment_artist_claims.profile_id AND l.track_id=a.track_id
        UNION ALL
        SELECT 1 FROM occurrence_credits a CROSS JOIN listens l
        WHERE a.artist_id=enrichment_artist_claims.artist_id AND a.occurrence_id<>OLD.id
          AND l.profile_id=enrichment_artist_claims.profile_id AND l.track_id=a.occurrence_id
    );
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE enrichment_artist_scope_downgrade_guard (restore_verified_backup INTEGER CHECK(restore_verified_backup=0));
INSERT INTO enrichment_artist_scope_downgrade_guard VALUES (1);
