-- +goose Up
CREATE TABLE account_tokens (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    token_hash BLOB NOT NULL CHECK (length(token_hash) = 32),
    purpose TEXT NOT NULL CHECK (purpose IN ('activation', 'reset')),
    expires_at INTEGER NOT NULL
);

-- +goose Down
-- Restore the verified snapshot for an intentional downgrade.
CREATE TEMP TABLE account_token_downgrade_guard(value INTEGER);
-- +goose StatementBegin
CREATE TEMP TRIGGER account_token_downgrade_block BEFORE INSERT ON account_token_downgrade_guard
BEGIN
    SELECT RAISE(ABORT,'Account token migration cannot be downgraded: restore backup');
END;
-- +goose StatementEnd
INSERT INTO account_token_downgrade_guard VALUES(1);
