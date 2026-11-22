-- +goose Up
-- Preserve key identifiers across deletion, even when the highest ID is removed.
CREATE TABLE api_keys_inbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    user_id INTEGER,
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    label TEXT NOT NULL DEFAULT ''
);
INSERT INTO api_keys_inbox SELECT id,key,user_id,profile_id,created_at,label FROM api_keys;
DROP TABLE api_keys;
ALTER TABLE api_keys_inbox RENAME TO api_keys;
CREATE INDEX idx_api_keys_profile_id ON api_keys(profile_id);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- +goose StatementBegin
CREATE TRIGGER trg_api_key_profile_immutable
BEFORE UPDATE OF profile_id ON api_keys
WHEN NEW.profile_id != OLD.profile_id
BEGIN
    SELECT RAISE(ABORT, 'API keys cannot be transferred between profiles');
END;
-- +goose StatementEnd

-- No foreign key: receipts retain attribution when their key is removed.
ALTER TABLE ingest_events ADD COLUMN api_key_id INTEGER;
ALTER TABLE ingest_events ADD COLUMN inbox_key INTEGER GENERATED ALWAYS AS (
    CASE WHEN source_type='file' THEN -1 ELSE coalesce(api_key_id,0) END
) VIRTUAL;
-- The index materializes classifications for old receipts and computes them on admission.
ALTER TABLE ingest_events ADD COLUMN inbox_metadata INTEGER GENERATED ALWAYS AS (
    CASE
        WHEN NOT json_valid(original_snapshot) THEN 1
        WHEN json_type(original_snapshot,'$.title') IS NOT 'text'
          OR json_type(original_snapshot,'$.artist') IS NOT 'text'
          OR trim(json_extract(original_snapshot,'$.title'))=''
          OR trim(json_extract(original_snapshot,'$.artist'))='' THEN 1
        WHEN trim(coalesce(json_extract(original_snapshot,'$.release'),''))=''
          OR coalesce(nullif(json_extract(original_snapshot,'$.recording_mbid'),''),nullif(json_extract(original_snapshot,'$.mapped_recording_mbid'),'')) IS NULL
          OR coalesce(nullif(json_extract(original_snapshot,'$.release_mbid'),''),nullif(json_extract(original_snapshot,'$.mapped_release_mbid'),'')) IS NULL
          OR (coalesce(json_array_length(original_snapshot,'$.artist_mbids'),0)=0
              AND coalesce(json_array_length(original_snapshot,'$.mapped_artist_mbids'),0)=0
              AND coalesce(json_array_length(original_snapshot,'$.artist_mbid_mappings'),0)=0)
          OR coalesce(CAST(json_extract(original_snapshot,'$.duration_ms') AS INTEGER),0)<=0 THEN 2
        ELSE 0
    END
) VIRTUAL;
CREATE INDEX idx_ingest_inbox_edges ON ingest_events(profile_id,received_at,inbox_key,health_client,state,inbox_metadata);
CREATE INDEX idx_ingest_inbox_example ON ingest_events(profile_id,inbox_key,health_client,inbox_metadata,received_at DESC,receipt_id DESC)
    WHERE inbox_metadata IN (1,2);

-- Count complete UTC days incrementally. The query reads individual receipts only at window boundaries.
CREATE TABLE ingest_inbox_daily (
    profile_id INTEGER NOT NULL REFERENCES listening_profiles(id) ON DELETE CASCADE,
    day INTEGER NOT NULL,
    key_id INTEGER NOT NULL,
    client TEXT NOT NULL,
    state TEXT NOT NULL,
    metadata INTEGER NOT NULL,
    total INTEGER NOT NULL CHECK(total>=0),
    PRIMARY KEY(profile_id,day,key_id,client,state,metadata)
) WITHOUT ROWID;

INSERT INTO ingest_inbox_daily
SELECT profile_id,received_at/86400,inbox_key,health_client,state,inbox_metadata,count(*)
FROM ingest_events INDEXED BY idx_ingest_inbox_edges
GROUP BY profile_id,received_at/86400,inbox_key,health_client,state,inbox_metadata;

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_inbox_insert AFTER INSERT ON ingest_events
BEGIN
    INSERT INTO ingest_inbox_daily VALUES(NEW.profile_id,NEW.received_at/86400,NEW.inbox_key,NEW.health_client,NEW.state,NEW.inbox_metadata,1)
    ON CONFLICT(profile_id,day,key_id,client,state,metadata) DO UPDATE SET total=total+1;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_inbox_delete AFTER DELETE ON ingest_events
BEGIN
    UPDATE ingest_inbox_daily SET total=total-1
    WHERE (profile_id,day,key_id,client,state,metadata)=(OLD.profile_id,OLD.received_at/86400,OLD.inbox_key,OLD.health_client,OLD.state,OLD.inbox_metadata);
    DELETE FROM ingest_inbox_daily
    WHERE (profile_id,day,key_id,client,state,metadata)=(OLD.profile_id,OLD.received_at/86400,OLD.inbox_key,OLD.health_client,OLD.state,OLD.inbox_metadata) AND total=0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_ingest_inbox_update AFTER UPDATE ON ingest_events
WHEN (OLD.profile_id,OLD.received_at/86400,OLD.inbox_key,OLD.health_client,OLD.state,OLD.inbox_metadata) IS NOT (NEW.profile_id,NEW.received_at/86400,NEW.inbox_key,NEW.health_client,NEW.state,NEW.inbox_metadata)
BEGIN
    UPDATE ingest_inbox_daily SET total=total-1
    WHERE (profile_id,day,key_id,client,state,metadata)=(OLD.profile_id,OLD.received_at/86400,OLD.inbox_key,OLD.health_client,OLD.state,OLD.inbox_metadata);
    DELETE FROM ingest_inbox_daily
    WHERE (profile_id,day,key_id,client,state,metadata)=(OLD.profile_id,OLD.received_at/86400,OLD.inbox_key,OLD.health_client,OLD.state,OLD.inbox_metadata) AND total=0;
    INSERT INTO ingest_inbox_daily VALUES(NEW.profile_id,NEW.received_at/86400,NEW.inbox_key,NEW.health_client,NEW.state,NEW.inbox_metadata,1)
    ON CONFLICT(profile_id,day,key_id,client,state,metadata) DO UPDATE SET total=total+1;
END;
-- +goose StatementEnd

-- +goose Down
CREATE TEMP TABLE ingest_inbox_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO ingest_inbox_downgrade_guard VALUES (1);
