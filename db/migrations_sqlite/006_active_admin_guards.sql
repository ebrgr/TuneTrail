-- +goose Up
-- Only administrators able to authenticate count towards the safety invariant.
DROP TRIGGER trg_last_admin_delete;
DROP TRIGGER trg_last_admin_demote;
-- +goose StatementBegin
CREATE TRIGGER trg_last_admin_delete BEFORE DELETE ON users
WHEN OLD.role='admin' AND OLD.status='active' AND (SELECT COUNT(*) FROM users WHERE role='admin' AND status='active')<=1
BEGIN
    SELECT RAISE(ABORT,'Last administrator is required');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_last_admin_demote BEFORE UPDATE OF role,status ON users
WHEN OLD.role='admin' AND OLD.status='active' AND (NEW.role<>'admin' OR NEW.status<>'active') AND (SELECT COUNT(*) FROM users WHERE role='admin' AND status='active')<=1
BEGIN
    SELECT RAISE(ABORT,'Last administrator is required');
END;
-- +goose StatementEnd

-- +goose Down
-- Restore a verified backup rather than weakening the active-administrator guard.
CREATE TEMP TABLE active_admin_downgrade_guard(value INTEGER);
-- +goose StatementBegin
CREATE TEMP TRIGGER active_admin_downgrade_block BEFORE INSERT ON active_admin_downgrade_guard
BEGIN
    SELECT RAISE(ABORT,'Active administrator migration cannot be downgraded: restore backup');
END;
-- +goose StatementEnd
INSERT INTO active_admin_downgrade_guard VALUES(1);
