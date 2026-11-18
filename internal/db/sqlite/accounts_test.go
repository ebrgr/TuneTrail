package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestAccountsMigrationPreservesData exercises both source layouts and committed WAL.
func TestAccountsMigrationPreservesData(t *testing.T) {
	for _, version := range []int64{3, 4} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			ctx := context.Background()
			s, _, source := migrationDB(t, version)
			hash, err := accountPassword("original", "changeme")
			require.NoError(t, err)
			execSQL(t, s, `INSERT INTO users(id,username,role,password) VALUES(7,'original','admin',?)`, hash)
			seedMusic(t, s)
			execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(20,'second','Second',123)`)
			execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(1,1,42,'original'),(20,1,42,'other')`)
			execSQL(t, s, `INSERT INTO api_keys(id,key,profile_id,created_at) VALUES(9,'keep',1,12),(10,'keep-other',20,13)`)
			execSQL(t, s, `INSERT INTO sessions VALUES('keep-session',7,0,2000000000,0); INSERT INTO profile_relays VALUES(1,'https://relay.test','keep-relay'); INSERT INTO legacy_relay_binding VALUES(1,1)`)
			if version == 4 {
				execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('preserve',1,'test',0)`)
			}
			before := legacyDigests(t, s.db)
			path := filepath.Join(t.TempDir(), "tunetrail.db")
			require.NoError(t, copyDatabase(source, path))
			conn, err := openExisting(path)
			require.NoError(t, err)
			defer conn.Close()
			execSQL(t, &Sqlite{db: conn}, `PRAGMA journal_mode=WAL; INSERT INTO listens(profile_id,track_id,listened_at) VALUES(20,1,43)`)
			before = legacyDigests(t, conn)
			_, err = bootstrap(filepath.Dir(path), true)
			require.NoError(t, err)
			upgraded := &Sqlite{db: conn}
			require.NoError(t, checkDatabase(conn))
			for name, want := range before {
				if name == "users" || name == "legacy_user_profiles" {
					continue
				}
				require.Equal(t, want, digestTable(t, conn, name, want.columns), name)
			}
			var kept []byte
			require.NoError(t, conn.QueryRow(`SELECT password FROM users WHERE id=7 AND role='admin' AND status='active'`).Scan(&kept))
			require.Equal(t, hash, kept)
			require.Equal(t, 1, scalar(t, upgraded, `SELECT COUNT(*) FROM sessions WHERE id='keep-session' AND user_id=7`))
			require.Equal(t, 2, scalar(t, upgraded, `SELECT COUNT(*) FROM users`))
			require.Equal(t, 1, scalar(t, upgraded, `SELECT COUNT(*) FROM users u JOIN listening_profiles p ON p.user_id=u.id WHERE p.id=20 AND status='pending' AND role='user' AND length(password)=0 AND require_login=0`))
			require.Zero(t, scalar(t, upgraded, `SELECT COUNT(*) FROM sqlite_master WHERE name IN ('trg_legacy_user_profile','legacy_user_profiles')`))
			// The saved source is restored into a separate directory before it is migrated again.
			pattern := ".pre-ingest-*.bak"
			if version == 4 {
				pattern = ".pre-accounts-*.bak"
			}
			backups, err := filepath.Glob(path + pattern)
			require.NoError(t, err)
			require.Len(t, backups, 1)
			restore := filepath.Join(t.TempDir(), "tunetrail.db")
			require.NoError(t, copyDatabase(backups[0], restore))
			restored, err := openExisting(restore)
			require.NoError(t, err)
			v, err := schemaVersion(restored)
			require.NoError(t, err)
			require.EqualValues(t, version, v)
			for name, want := range before {
				require.Equal(t, want, digestTable(t, restored, name, want.columns))
			}
			require.NoError(t, restored.Close())
			_, err = bootstrap(filepath.Dir(restore), true)
			require.NoError(t, err)
			stable := legacyDigests(t, conn)
			_, err = bootstrap(filepath.Dir(path), true)
			require.NoError(t, err)
			require.Equal(t, stable, legacyDigests(t, conn))
			_, err = upgraded.SaveAuthenticatedSession(ctx, 7, hash, time.Now().Add(time.Hour), false)
			require.NoError(t, err)
		})
	}
}

// TestAccountsCollisionRollsBack proves a real SQL error restores DDL, mappings and migration history.
func TestAccountsCollisionRollsBack(t *testing.T) {
	s, p, _ := migrationDB(t, 4)
	execSQL(t, s, `INSERT INTO users(id,username,role,password) VALUES(1,'taken','admin',X'01'),(2,'profile-20','user',X'02'); INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(20,'taken','Pending',0)`)
	before := legacyDigests(t, s.db)
	_, err := p.Up(context.Background())
	require.Error(t, err)
	require.Equal(t, 4, scalar(t, s, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1`))
	require.NoError(t, checkDatabase(s.db))
	require.Equal(t, before, legacyDigests(t, s.db))
	execSQL(t, s, `UPDATE listening_profiles SET slug='available' WHERE id=20`)
	_, err = p.Up(context.Background())
	require.NoError(t, err)
	require.NoError(t, checkDatabase(s.db))
	_, err = p.Up(context.Background())
	require.NoError(t, err)
}

// TestAccountsMissingProfiles reconciles accounts without profiles without changing roles.
func TestAccountsMissingProfiles(t *testing.T) {
	s, p, _ := migrationDB(t, 4)
	execSQL(t, s, `INSERT INTO users(id,username,password) VALUES(10,'one',X'01'),(11,'two',X'02'); DELETE FROM listening_profiles`)
	_, err := p.Up(context.Background())
	require.NoError(t, err)
	require.NoError(t, checkDatabase(s.db))
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM listening_profiles`))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM users WHERE role='admin'`))
	created, err := s.BootstrapAdministrator(context.Background(), db.BootstrapAdminOpts{Username: "admin", Password: "changeme", APIKey: "key"})
	require.NoError(t, err)
	require.False(t, created)
}

// TestAccountConstraints verifies additive NOT NULL enforcement, ownership and activation states.
func TestAccountConstraints(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	created, err := s.BootstrapAdministrator(ctx, db.BootstrapAdminOpts{Username: "admin", Password: "changeme", APIKey: "key"})
	require.NoError(t, err)
	require.True(t, created)
	p, err := s.CreateListeningProfile(ctx, "Pending", "pending", "", "pending-key")
	require.NoError(t, err)
	for _, q := range []string{
		`INSERT INTO listening_profiles(slug,display_name,created_at) VALUES('missing','Missing',0)`,
		`INSERT INTO listening_profiles(user_id,slug,display_name,created_at) VALUES(999,'missing','Missing',0)`,
		`INSERT INTO listening_profiles(user_id,slug,display_name,created_at) VALUES(1,'duplicate','Duplicate',0)`,
		`UPDATE listening_profiles SET user_id=NULL`, `UPDATE listening_profiles SET user_id=2 WHERE id=1`,
		`UPDATE listening_profiles SET require_login=2`, `DELETE FROM users WHERE id=2`,
		`UPDATE users SET status='active' WHERE id=2`, `UPDATE users SET password=X'' WHERE id=1`,
		`DELETE FROM users WHERE id=1`, `UPDATE users SET role='user' WHERE id=1`,
	} {
		_, err = s.db.Exec(q)
		require.Error(t, err, q)
	}
	require.Zero(t, scalar(t, s, `SELECT SUM(require_login) FROM listening_profiles`))
	pending, err := s.GetUserByUsername(ctx, "pending")
	require.NoError(t, err)
	require.Equal(t, "pending", pending.Status)
	_, err = admin.Login(ctx, s, "pending", "anything", false)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	// Even an invalidly promoted pending account cannot authenticate using an empty hash.
	execSQL(t, s, `UPDATE users SET role='admin' WHERE id=?`, pending.ID)
	_, err = s.SaveAuthenticatedSession(ctx, pending.ID, []byte{}, time.Now().Add(time.Hour), false)
	require.ErrorIs(t, err, db.ErrConflict)
	require.NoError(t, s.UpdateUser(ctx, db.UpdateUserOpts{ID: pending.ID, Password: "activated-password"}))
	active, err := s.GetUserByUsername(ctx, "pending")
	require.NoError(t, err)
	require.Equal(t, "active", active.Status)
	require.NoError(t, bcrypt.CompareHashAndPassword(active.Password, []byte("activated-password")))
	require.NoError(t, s.DeleteListeningProfile(ctx, p.ID, "Pending", 0))
	require.NoError(t, checkDatabase(s.db))
}

// TestProfileCreationCollisionAndRollback covers truncation, bounded suffixes and failed key creation.
func TestProfileCreationCollisionAndRollback(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	base := strings.Repeat("a", 32)
	for n := 1; n <= 9; n++ {
		slug := base + fmt.Sprint(n)
		_, err = s.CreateListeningProfile(ctx, slug, slug, "", fmt.Sprint(n))
		require.NoError(t, err)
	}
	_, err = s.CreateListeningProfile(ctx, "Exhausted", base+"last", "", "last")
	require.ErrorIs(t, err, db.ErrConflict)
	require.Equal(t, 9, scalar(t, s, `SELECT COUNT(*) FROM users`))
	_, err = s.CreateListeningProfile(ctx, "Bad key", "bad-key", "", "1")
	require.Error(t, err)
	require.Equal(t, 9, scalar(t, s, `SELECT COUNT(*) FROM users`))
	require.NoError(t, checkDatabase(s.db))
}

// TestBootstrapOriginalV2Defaults preserves configured bootstrap after migration, without extra users.
func TestBootstrapOriginalV2Defaults(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			s, _, source := migrationDB(t, 2)
			if existing {
				execSQL(t, s, `INSERT INTO users(id,username,role,password) VALUES(9,'existing','admin',X'01')`)
			}
			require.NoError(t, cfg.Load(func(string) string { return "" }, "test"))
			require.Equal(t, "admin", cfg.DefaultUsername())
			require.Equal(t, "changeme", cfg.DefaultPassword())
			path, err := bootstrap(filepath.Dir(source), true)
			require.NoError(t, err)
			conn, err := openExisting(path)
			require.NoError(t, err)
			defer conn.Close()
			upgraded := &Sqlite{db: conn}
			created, err := upgraded.BootstrapAdministrator(context.Background(), db.BootstrapAdminOpts{Username: cfg.DefaultUsername(), Password: cfg.DefaultPassword(), APIKey: "default-key"})
			require.NoError(t, err)
			require.Equal(t, !existing, created)
			require.Equal(t, 1, scalar(t, upgraded, `SELECT COUNT(*) FROM users`))
			require.NoError(t, checkDatabase(conn))
			if !existing {
				u, err := upgraded.GetUserByUsername(context.Background(), "admin")
				require.NoError(t, err)
				require.Equal(t, models.UserRoleAdmin, u.Role)
				require.Equal(t, "active", u.Status)
				require.NoError(t, bcrypt.CompareHashAndPassword(u.Password, []byte("changeme")))
				require.Equal(t, 1, scalar(t, upgraded, `SELECT COUNT(*) FROM api_keys k JOIN listening_profiles p ON p.id=k.profile_id WHERE p.user_id=1 AND k.label='Default'`))
			}
		})
	}
}

// TestConcurrentAccountMutations serializes bootstrap and deletion through SQLite write locks.
func TestConcurrentAccountMutations(t *testing.T) {
	path, err := bootstrap(t.TempDir(), true)
	require.NoError(t, err)
	a, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	defer a.Close()
	s := &Sqlite{db: a}
	ctx := context.Background()
	opts := db.BootstrapAdminOpts{Username: "admin", Password: "changeme", APIKey: "key"}
	results := make(chan bool, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, e := s.BootstrapAdministrator(ctx, opts)
			results <- created
			errs <- e
		}()
	}
	wg.Wait()
	count := 0
	for range 2 {
		require.NoError(t, <-errs)
		if <-results {
			count++
		}
	}
	require.Equal(t, 1, count)
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "second-admin", Password: "changeme", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	for _, id := range []int32{1, 2} {
		wg.Add(1)
		go func(id int32) {
			defer wg.Done()
			name := "admin"
			if id == 2 {
				name = "second-admin"
			}
			errs <- s.DeleteListeningProfile(ctx, id, name, 0)
		}(id)
	}
	wg.Wait()
	one, two := <-errs, <-errs
	require.True(t, (one == nil && two == db.ErrLastAdmin) || (two == nil && one == db.ErrLastAdmin), fmt.Sprint(one, two))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM users WHERE role='admin'`))
	require.NoError(t, checkDatabase(a))
}

// TestAccountPairWriteRollback rejects partial bootstrap and failed account/profile writes.
func TestAccountPairWriteRollback(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	execSQL(t, s, `CREATE TRIGGER fail_initial_key BEFORE INSERT ON api_keys BEGIN SELECT RAISE(ABORT,'injected key failure'); END`)
	created, err := s.BootstrapAdministrator(ctx, db.BootstrapAdminOpts{Username: "admin", Password: "changeme", APIKey: "key"})
	require.Error(t, err)
	require.False(t, created)
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM users`))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM listening_profiles`))
	execSQL(t, s, `DROP TRIGGER fail_initial_key; CREATE TRIGGER fail_profile BEFORE INSERT ON listening_profiles BEGIN SELECT RAISE(ABORT,'injected profile failure'); END`)
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "test", Password: "changeme"})
	require.Error(t, err)
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM users`))
}

// TestConcurrentAdminDemotions ensures the database guard protects the last role across writers.
func TestConcurrentAdminDemotions(t *testing.T) {
	path, err := bootstrap(t.TempDir(), true)
	require.NoError(t, err)
	conn, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	defer conn.Close()
	s := &Sqlite{db: conn}
	for _, name := range []string{"one", "two"} {
		_, err = s.SaveUser(context.Background(), db.SaveUserOpts{Username: name, Password: "changeme", Role: models.UserRoleAdmin})
		require.NoError(t, err)
	}
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, id := range []int{1, 2} {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, e := conn.Exec(`UPDATE users SET role='user' WHERE id=?`, id)
			errs <- e
		}(id)
	}
	wg.Wait()
	one, two := <-errs, <-errs
	require.True(t, (one == nil) != (two == nil))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM users WHERE role='admin'`))
	require.NoError(t, checkDatabase(conn))
}

// TestAccountsIncompleteLayout refuses missing guards and broken bidirectional cardinality.
func TestAccountsIncompleteLayout(t *testing.T) {
	for _, damage := range []string{
		`INSERT INTO users(username,password) VALUES('orphan',X'01')`,
		`DROP TRIGGER trg_profile_owner_required`,
		`DROP INDEX idx_listening_profiles_user_id`,
		`DROP TRIGGER trg_profile_owner_immutable; UPDATE listening_profiles SET user_id=NULL`,
	} {
		t.Run(damage, func(t *testing.T) {
			s, err := NewInMemory()
			require.NoError(t, err)
			defer s.db.Close()
			_, err = s.BootstrapAdministrator(context.Background(), db.BootstrapAdminOpts{Username: "admin", Password: "changeme", APIKey: "key"})
			require.NoError(t, err)
			execSQL(t, s, damage)
			require.Error(t, checkDatabase(s.db))
		})
	}
}
