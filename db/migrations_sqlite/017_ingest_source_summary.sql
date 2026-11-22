-- +goose Up
-- Keep the receipt identity separate from the source dimensions so a key dimension
-- can be added independently of how delivery state is read.
ALTER TABLE ingest_events ADD COLUMN health_source TEXT GENERATED ALWAYS AS (coalesce(file_format,source_type)) VIRTUAL;
ALTER TABLE ingest_events ADD COLUMN health_client TEXT GENERATED ALWAYS AS (
    substr(coalesce(nullif(client_name,''),CASE WHEN json_valid(original_snapshot) THEN
        CASE WHEN json_type(original_snapshot,'$.client')='text' THEN json_extract(original_snapshot,'$.client') END
    END,''),1,120)
) VIRTUAL;

CREATE TABLE ingest_source_summary (
    profile_id INTEGER NOT NULL,
    source_instance_id TEXT NOT NULL,
    source TEXT NOT NULL,
    client TEXT NOT NULL,
    received_at INTEGER NOT NULL,
    receipt_id TEXT NOT NULL,
    PRIMARY KEY(profile_id,source_instance_id,source,client),
    FOREIGN KEY(profile_id,source_instance_id) REFERENCES integrations(profile_id,id) ON DELETE CASCADE
) WITHOUT ROWID;

CREATE INDEX idx_ingest_source_summary_recent ON ingest_source_summary(profile_id,received_at DESC,receipt_id DESC);
CREATE INDEX idx_ingest_source_receipt ON ingest_events(
    profile_id,source_instance_id,health_source,health_client,received_at DESC,receipt_id DESC
);

-- The index streams the historical groups in their latest-receipt order.
INSERT INTO ingest_source_summary(profile_id,source_instance_id,source,client,received_at,receipt_id)
SELECT profile_id,source_instance_id,health_source,health_client,received_at,receipt_id FROM (
    SELECT profile_id,source_instance_id,health_source,health_client,received_at,receipt_id,
        row_number() OVER (
            PARTITION BY profile_id,source_instance_id,health_source,health_client ORDER BY received_at DESC,receipt_id DESC
        ) AS rank
    FROM ingest_events
) WHERE rank=1;

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_source_admit AFTER INSERT ON ingest_events
BEGIN
    INSERT INTO ingest_source_summary(profile_id,source_instance_id,source,client,received_at,receipt_id)
    VALUES(NEW.profile_id,NEW.source_instance_id,NEW.health_source,NEW.health_client,NEW.received_at,NEW.receipt_id)
    ON CONFLICT(profile_id,source_instance_id,source,client) DO UPDATE
        SET received_at=excluded.received_at,receipt_id=excluded.receipt_id
        WHERE (excluded.received_at,excluded.receipt_id)>(ingest_source_summary.received_at,ingest_source_summary.receipt_id);
END;
-- +goose StatementEnd

-- Forget deleted receipts and fall back to the previous receipt in the same group.
-- +goose StatementBegin
CREATE TRIGGER trg_ingest_source_delete AFTER DELETE ON ingest_events
BEGIN
    DELETE FROM ingest_source_summary
    WHERE (profile_id,source_instance_id,source,client,receipt_id)=
        (OLD.profile_id,OLD.source_instance_id,OLD.health_source,OLD.health_client,OLD.receipt_id)
        AND NOT EXISTS (
            SELECT 1 FROM ingest_events
            WHERE (profile_id,source_instance_id,health_source,health_client)=
                (OLD.profile_id,OLD.source_instance_id,OLD.health_source,OLD.health_client)
        );
    UPDATE ingest_source_summary SET (received_at,receipt_id)=(
        SELECT received_at,receipt_id FROM ingest_events
        WHERE (profile_id,source_instance_id,health_source,health_client)=
            (OLD.profile_id,OLD.source_instance_id,OLD.health_source,OLD.health_client)
        ORDER BY received_at DESC,receipt_id DESC LIMIT 1
    )
    WHERE (profile_id,source_instance_id,source,client,receipt_id)=
        (OLD.profile_id,OLD.source_instance_id,OLD.health_source,OLD.health_client,OLD.receipt_id);
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE ingest_source_summary_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO ingest_source_summary_downgrade_guard VALUES (1);
