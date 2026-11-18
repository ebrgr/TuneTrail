-- +goose Up
-- Keep parent tables in place: dropping either would cascade into listening data.
DROP TRIGGER trg_legacy_user_profile;
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','pending') AND ((status='pending') = (length(password)=0)));
ALTER TABLE listening_profiles ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE RESTRICT;
ALTER TABLE listening_profiles ADD COLUMN require_login INTEGER NOT NULL DEFAULT 0 CHECK (require_login IN (0,1));

UPDATE listening_profiles SET user_id=(SELECT user_id FROM legacy_user_profiles WHERE profile_id=listening_profiles.id);

-- Restore missing account profiles explicitly, never by matching display names.
INSERT INTO listening_profiles(slug,display_name,created_at,user_id)
SELECT CASE WHEN NOT EXISTS(SELECT 1 FROM listening_profiles) AND u.id=(SELECT MIN(id) FROM users)
            THEN 'default' ELSE 'user-' || u.id END, u.username,unixepoch(),u.id
FROM users u WHERE NOT EXISTS(SELECT 1 FROM listening_profiles p WHERE p.user_id=u.id);

-- A conflicting fallback is an operator reconciliation error, not a silent reassignment.
CREATE TEMP TABLE account_plan(profile_id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE);
INSERT INTO account_plan
SELECT p.id, CASE WHEN EXISTS(SELECT 1 FROM users WHERE username=substr(p.slug,1,32))
                   OR EXISTS(SELECT 1 FROM listening_profiles q WHERE q.user_id IS NULL AND q.id<p.id AND substr(q.slug,1,32)=substr(p.slug,1,32))
                  THEN 'profile-' || p.id ELSE substr(p.slug,1,32) END
FROM listening_profiles p WHERE p.user_id IS NULL;
INSERT INTO users(username,role,password,status) SELECT username,'user',X'','pending' FROM account_plan;
UPDATE listening_profiles SET user_id=(SELECT u.id FROM account_plan a JOIN users u ON u.username=a.username WHERE a.profile_id=listening_profiles.id) WHERE user_id IS NULL;

CREATE TEMP TABLE account_guard(value INTEGER);
-- +goose StatementBegin
CREATE TEMP TRIGGER account_guard_invalid BEFORE INSERT ON account_guard
WHEN EXISTS(SELECT 1 FROM listening_profiles WHERE user_id IS NULL)
  OR EXISTS(SELECT 1 FROM users u WHERE NOT EXISTS(SELECT 1 FROM listening_profiles p WHERE p.user_id=u.id))
BEGIN
    SELECT RAISE(ABORT,'Account/profile reconciliation required');
END;
-- +goose StatementEnd
INSERT INTO account_guard VALUES(1);
CREATE UNIQUE INDEX idx_listening_profiles_user_id ON listening_profiles(user_id);
-- ADD COLUMN cannot declare NOT NULL on existing rows; these guards enforce it without rebuilding parents.
-- +goose StatementBegin
CREATE TRIGGER trg_profile_owner_required BEFORE INSERT ON listening_profiles WHEN NEW.user_id IS NULL
BEGIN
    SELECT RAISE(ABORT,'Profile owner is required');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_profile_owner_immutable BEFORE UPDATE OF user_id ON listening_profiles
WHEN NEW.user_id IS NULL OR NEW.user_id IS NOT OLD.user_id
BEGIN
    SELECT RAISE(ABORT,'Profile owner cannot change');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_last_admin_delete BEFORE DELETE ON users
WHEN OLD.role='admin' AND (SELECT COUNT(*) FROM users WHERE role='admin')<=1
BEGIN
    SELECT RAISE(ABORT,'Last administrator is required');
END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_last_admin_demote BEFORE UPDATE OF role,status ON users
WHEN OLD.role='admin' AND (NEW.role<>'admin' OR NEW.status<>'active') AND (SELECT COUNT(*) FROM users WHERE role='admin')<=1
BEGIN
    SELECT RAISE(ABORT,'Last administrator is required');
END;
-- +goose StatementEnd
-- The relay binding already references stable profile IDs, and remains untouched.
DROP TABLE legacy_user_profiles;
DROP TABLE account_plan;
DROP TABLE account_guard;

-- +goose Down
-- Restoring a verified backup is the only supported downgrade.
CREATE TEMP TABLE accounts_downgrade_guard(value INTEGER);
-- +goose StatementBegin
CREATE TEMP TRIGGER accounts_downgrade_block BEFORE INSERT ON accounts_downgrade_guard
BEGIN
    SELECT RAISE(ABORT,'Accounts migration cannot be downgraded: restore the pre-accounts backup');
END;
-- +goose StatementEnd
INSERT INTO accounts_downgrade_guard VALUES(1);
