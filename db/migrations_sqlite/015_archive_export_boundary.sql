-- +goose Up
-- Sequence values cannot be reused when the newest listen is deleted.
CREATE TABLE listen_export_order (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL UNIQUE REFERENCES listens(event_id) ON DELETE CASCADE
);
INSERT INTO listen_export_order(event_id) SELECT event_id FROM listens ORDER BY rowid;

-- +goose StatementBegin
CREATE TRIGGER trg_listen_export_order AFTER INSERT ON listens
BEGIN
    INSERT INTO listen_export_order(event_id) VALUES (NEW.event_id);
END;
-- +goose StatementEnd

-- Existing undo records start their retention window at upgrade time.
ALTER TABLE recording_moves ADD COLUMN expires_at INTEGER NOT NULL DEFAULT 0;
UPDATE recording_moves SET expires_at=unixepoch()+2592000 WHERE stage=1;
CREATE INDEX idx_recording_moves_expiry ON recording_moves(expires_at) WHERE stage=1;

-- +goose Down
CREATE TEMP TABLE archive_export_boundary_downgrade_guard (ok INTEGER CHECK(ok=0));
INSERT INTO archive_export_boundary_downgrade_guard VALUES (1);
