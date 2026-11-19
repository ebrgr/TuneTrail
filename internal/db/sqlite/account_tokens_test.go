package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// TestAccountTokenTransaction162 verifies activation expiry and rollback across all writes.
func TestAccountTokenTransaction162(t *testing.T) {
	ctx := context.Background()
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "root-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	session, err := admin.Login(ctx, s, root.Username, "root-password", false)
	require.NoError(t, err)
	// A key insertion failure must leave neither the pending account nor its profile.
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_key BEFORE INSERT ON api_keys BEGIN SELECT RAISE(ABORT,'injected failure'); END`))
	_, err = s.CreateListeningProfile(ctx, "Fail", "fail", "", "fail-key")
	require.Error(t, err)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM users`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listening_profiles`))
	require.NoError(t, s.Exec(`DROP TRIGGER fail_key`))
	p, err := s.CreateListeningProfile(ctx, "Pending", "pending", "", "pending-key")
	require.NoError(t, err)
	digest := sha256.Sum256([]byte("fictional activation digest"))
	require.NoError(t, s.ReplaceAccountToken(ctx, session.ID, p.ID, digest))
	var saved []byte
	var purpose string
	var ttl int
	require.NoError(t, s.db.QueryRow(`SELECT token_hash,purpose,expires_at-unixepoch() FROM account_tokens`).Scan(&saved, &purpose, &ttl))
	require.Equal(t, digest[:], saved)
	require.Equal(t, "activation", purpose)
	require.InDelta(t, 86400, ttl, 3)
	// An invalid credential must be rejected before bcrypt can reject the oversized password.
	oversized := strings.Repeat("x", 73)
	wrong := sha256.Sum256([]byte("different fictional token"))
	require.ErrorIs(t, s.RedeemAccountToken(ctx, "pending", wrong, oversized), db.ErrAccountToken)
	require.ErrorIs(t, s.RedeemAccountToken(ctx, "pending", digest, oversized), db.ErrInvalidInput)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	require.NoError(t, s.Exec(`UPDATE account_tokens SET expires_at=unixepoch()`))
	require.ErrorIs(t, s.RedeemAccountToken(ctx, "pending", digest, "new-password"), db.ErrAccountToken)
	require.NoError(t, s.ReplaceAccountToken(ctx, session.ID, p.ID, digest))
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_password BEFORE UPDATE OF password ON users BEGIN SELECT RAISE(ABORT,'injected failure'); END`))
	require.Error(t, s.RedeemAccountToken(ctx, "pending", digest, "new-password"))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM users WHERE status='pending' AND length(password)=0`))
	require.NoError(t, s.Exec(`DROP TRIGGER fail_password`))
	require.NoError(t, s.RedeemAccountToken(ctx, "pending", digest, "new-password"))
	require.ErrorIs(t, s.RedeemAccountToken(ctx, "pending", digest, "new-password"), db.ErrAccountToken)
	pendingSession, err := admin.Login(ctx, s, "pending", "new-password", false)
	require.NoError(t, err)
	require.NoError(t, s.ReplaceAccountToken(ctx, session.ID, p.ID, digest))
	require.NoError(t, s.db.QueryRow(`SELECT purpose,expires_at-unixepoch() FROM account_tokens`).Scan(&purpose, &ttl))
	require.Equal(t, "reset", purpose)
	require.InDelta(t, 1800, ttl, 3)
	seedMusic(t, s)
	require.NoError(t, s.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,100),(?,1,100)`, p.ID))
	require.NoError(t, s.SetProfileRelay(ctx, p.ID, "https://relay.invalid", "pending-relay"))
	require.NoError(t, s.SetProfileRelay(ctx, 1, "https://other.invalid", "other-relay"))
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_user_delete BEFORE DELETE ON users WHEN OLD.username='pending' BEGIN SELECT RAISE(ABORT,'injected failure'); END`))
	require.Error(t, s.DeleteListeningProfileAsRoot(ctx, session.ID, p.ID, p.Slug, p.DisplayName, 1))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	_, err = s.GetProfileByAPIKey(ctx, "pending-key")
	require.NoError(t, err)
	_, err = admin.Authenticate(ctx, s, pendingSession.ID)
	require.NoError(t, err)
	require.NoError(t, s.Exec(`DROP TRIGGER fail_user_delete`))
	require.NoError(t, s.DeleteListeningProfileAsRoot(ctx, session.ID, p.ID, p.Slug, p.DisplayName, 1))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM users`))
	_, err = s.GetProfileByAPIKey(ctx, "pending-key")
	require.ErrorIs(t, err, db.ErrNotFound)
	_, err = admin.Authenticate(ctx, s, pendingSession.ID)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	_, err = admin.Authenticate(ctx, s, session.ID)
	require.NoError(t, err)
	_, relay, err := s.GetProfileRelay(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, "other-relay", relay)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM profile_relays`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks`))
}

// TestAccountTokenAccessChanges162 covers purpose binding and invalidation on access changes.
func TestAccountTokenAccessChanges162(t *testing.T) {
	ctx := context.Background()
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "root-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	session, err := admin.Login(ctx, s, "root", "root-password", false)
	require.NoError(t, err)
	target, err := s.CreateListeningProfile(ctx, "Target", "target", "", "target-key")
	require.NoError(t, err)
	digest := sha256.Sum256([]byte("fictional token for changed privileges"))
	issue := func() {
		t.Helper()
		require.NoError(t, s.ReplaceAccountToken(ctx, session.ID, target.ID, digest))
	}
	rejected := func() {
		t.Helper()
		require.ErrorIs(t, s.RedeemAccountToken(ctx, "target", digest, "different-password"), db.ErrAccountToken)
	}
	issue()
	// A pending account cannot redeem a recovery token, even with a matching unexpired digest.
	require.NoError(t, s.Exec(`UPDATE account_tokens SET purpose='reset' WHERE user_id=?`, target.UserID))
	rejected()
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	issue()
	// Match the existing local password change, which makes the pending activation stale.
	require.NoError(t, s.UpdateUser(ctx, db.UpdateUserOpts{ID: target.UserID, Password: "local-password"}))
	rejected()
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
	for _, change := range []struct {
		name   string
		role   models.UserRole
		status string
	}{
		{"promotion", models.UserRoleAdmin, "active"},
		{"demotion", models.UserRoleUser, "active"},
		{"deactivation", models.UserRoleUser, "pending"},
	} {
		t.Run(change.name, func(t *testing.T) {
			issue()
			require.NoError(t, s.UpdateAccountAccess(ctx, session.ID, target.UserID, change.role, change.status))
			require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM account_tokens`))
			rejected()
		})
	}
}

// TestAccountTokenUpgrade162 verifies backup/restart and refuses a claimed v7 without its table.
func TestAccountTokenUpgrade162(t *testing.T) {
	s, _, source := migrationDB(t, 6)
	ctx := context.Background()
	_, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "root-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	require.NoError(t, s.db.Close())
	canonical := filepath.Join(filepath.Dir(source), "tunetrail.db")
	require.NoError(t, os.Rename(source, canonical))
	source = canonical
	upgraded, err := bootstrap(filepath.Dir(source), false)
	require.NoError(t, err)
	verify, err := sql.Open("sqlite", sqliteURI(upgraded))
	require.NoError(t, err)
	defer verify.Close()
	backup, err := filepath.Glob(source + ".pre-account-tokens-*.bak")
	require.NoError(t, err)
	require.Len(t, backup, 1)
	old, err := sql.Open("sqlite", sqliteURI(backup[0]))
	require.NoError(t, err)
	defer old.Close()
	require.NoError(t, checkDatabase(old))
	version, err := schemaVersion(old)
	require.NoError(t, err)
	require.Equal(t, 6, version)
	version, err = schemaVersion(verify)
	require.NoError(t, err)
	require.Equal(t, currentSchemaVersion, version)
	require.NoError(t, checkDatabase(verify))
	_, err = bootstrap(filepath.Dir(source), false)
	require.NoError(t, err)
	_, err = verify.Exec(`DROP TABLE account_tokens`)
	require.NoError(t, err)
	require.ErrorContains(t, checkDatabase(verify), "account_tokens")
}

// TestAccountTokenWAL162 checks single consumption and competing confirmed self-deletions.
func TestAccountTokenWAL162(t *testing.T) {
	ctx := context.Background()
	path, err := bootstrap(t.TempDir(), true)
	require.NoError(t, err)
	var stores [2]*Sqlite
	var profiles [2]*models.ListeningProfile
	var sessions [2]*models.Session
	for i, name := range []string{"first", "second"} {
		conn, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
		require.NoError(t, err)
		defer conn.Close()
		s := &Sqlite{db: conn}
		stores[i] = s
		u, err := s.SaveUser(ctx, db.SaveUserOpts{Username: name, Password: "root-password", Role: models.UserRoleAdmin})
		require.NoError(t, err)
		var id int32
		require.NoError(t, conn.QueryRow(`SELECT id FROM listening_profiles WHERE user_id=?`, u.ID).Scan(&id))
		profiles[i], err = s.GetListeningProfileByID(ctx, id)
		require.NoError(t, err)
		sessions[i], err = admin.Login(ctx, s, name, "root-password", false)
		require.NoError(t, err)
	}
	var journal string
	require.NoError(t, stores[0].db.QueryRow(`PRAGMA journal_mode`).Scan(&journal))
	require.Equal(t, "wal", journal)
	digest := sha256.Sum256([]byte("fictional concurrent token"))
	require.NoError(t, stores[0].ReplaceAccountToken(ctx, sessions[0].ID, profiles[1].ID, digest))
	race := func(op func(int) error, rejected error) int {
		t.Helper()
		type result struct {
			index int
			err   error
		}
		start := make(chan struct{})
		done := make(chan result, 2)
		for i := range stores {
			go func() { <-start; done <- result{i, op(i)} }()
		}
		close(start)
		winner := -1
		for range stores {
			got := <-done
			if got.err == nil {
				require.Equal(t, -1, winner)
				winner = got.index
			} else {
				require.ErrorIs(t, got.err, rejected)
			}
		}
		require.NotEqual(t, -1, winner)
		return winner
	}
	race(func(i int) error {
		return stores[i].RedeemAccountToken(ctx, "second", digest, "reset-password")
	}, db.ErrAccountToken)
	sessions[1], err = admin.Login(ctx, stores[1], "second", "reset-password", false)
	require.NoError(t, err)
	winner := race(func(i int) error {
		s, p := stores[i], profiles[i]
		// Compete across independent SQLite connections without the service mutex.
		return s.DeleteListeningProfileAsRoot(ctx, sessions[i].ID, p.ID, p.Slug, p.DisplayName, 0)
	}, db.ErrLastAdmin)
	_, err = admin.Authenticate(ctx, stores[winner], sessions[winner].ID)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	_, err = admin.Authenticate(ctx, stores[1-winner], sessions[1-winner].ID)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, stores[0], `SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`))
	require.NoError(t, checkDatabase(stores[0].db))
	require.True(t, sessions[1-winner].ExpiresAt.After(time.Now()))
}
