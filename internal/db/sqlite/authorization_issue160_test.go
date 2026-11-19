package sqlite

import (
	"context"
	"database/sql"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestLastActiveAdministrator160 includes pending admins and independent SQLite
// connections, so serialization cannot accidentally depend on a process mutex.
func TestLastActiveAdministrator160(t *testing.T) {
	for _, mode := range []string{"demote", "deactivate", "delete", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			path, err := bootstrap(t.TempDir(), true)
			require.NoError(t, err)
			a, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
			require.NoError(t, err)
			defer a.Close()
			b, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
			require.NoError(t, err)
			defer b.Close()
			stores := []*Sqlite{{db: a}, {db: b}}
			var journal string
			require.NoError(t, a.QueryRow(`PRAGMA journal_mode`).Scan(&journal))
			require.Equal(t, "wal", journal)
			var users []*models.User
			var sessions []*models.Session
			for _, name := range []string{"one", "two"} {
				u, err := stores[0].SaveUser(ctx, db.SaveUserOpts{Username: name, Password: "test-password", Role: models.UserRoleAdmin})
				require.NoError(t, err)
				users = append(users, u)
				session, err := admin.Login(ctx, stores[0], name, "test-password", false)
				require.NoError(t, err)
				sessions = append(sessions, session)
			}
			pending, err := stores[0].CreateListeningProfile(ctx, "Pending", "pending", "", "pending-key")
			require.NoError(t, err)
			require.NoError(t, stores[0].Exec(`UPDATE users SET role='admin' WHERE id=?`, pending.UserID))
			start := make(chan struct{})
			results := make(chan error, 2)
			var wg sync.WaitGroup
			for i := range 2 {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					op := mode
					if mode == "mixed" {
						if i == 0 {
							op = "delete"
						} else {
							op = "deactivate"
						}
					}
					if op == "delete" {
						profiles, err := stores[i].ListListeningProfiles(ctx)
						if err != nil {
							results <- err
							return
						}
						for _, p := range profiles {
							if p.UserID == users[i].ID {
								results <- stores[i].DeleteListeningProfile(ctx, p.ID, p.DisplayName, 0)
								return
							}
						}
						results <- db.ErrNotFound
						return
					}
					role, status := models.UserRoleUser, "active"
					if op == "deactivate" {
						role, status = models.UserRoleAdmin, "pending"
					}
					results <- stores[i].UpdateAccountAccess(ctx, sessions[i].ID, users[i].ID, role, status)
				}(i)
			}
			close(start)
			wg.Wait()
			success := 0
			for range 2 {
				err := <-results
				if err == nil {
					success++
				} else {
					require.ErrorIs(t, err, db.ErrLastAdmin)
				}
			}
			require.Equal(t, 1, success)
			require.Equal(t, 1, scalar(t, stores[0], `SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`))
			require.NoError(t, checkDatabase(a))
		})
	}
}

// TestAccessChangesUseCurrentSession160 rejects promotion by a user and stale root sessions.
func TestAccessChangesUseCurrentSession160(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	user, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "user", Password: "test-password"})
	require.NoError(t, err)
	rootSession, err := admin.Login(ctx, s, "root", "test-password", false)
	require.NoError(t, err)
	userSession, err := admin.Login(ctx, s, "user", "test-password", false)
	require.NoError(t, err)
	require.ErrorIs(t, admin.ChangeAccountAccess(ctx, s, userSession.ID, user.ID, models.UserRoleAdmin, "active"), admin.ErrForbidden)
	require.ErrorIs(t, s.UpdateAccountAccess(ctx, userSession.ID, user.ID, models.UserRoleAdmin, "active"), db.ErrConflict)
	require.NoError(t, admin.ChangeAccountAccess(ctx, s, rootSession.ID, user.ID, models.UserRoleAdmin, "active"))
	require.NoError(t, admin.ChangeAccountAccess(ctx, s, rootSession.ID, root.ID, models.UserRoleUser, "active"))
	require.ErrorIs(t, admin.ChangeAccountAccess(ctx, s, rootSession.ID, root.ID, models.UserRoleAdmin, "active"), admin.ErrForbidden)
	require.ErrorIs(t, s.UpdateAccountAccess(ctx, rootSession.ID, root.ID, models.UserRoleAdmin, "active"), db.ErrConflict)
	require.ErrorIs(t, admin.ChangeAccountAccess(ctx, s, userSession.ID, user.ID, models.UserRoleUser, "active"), db.ErrLastAdmin)
	require.NoError(t, s.Exec(`UPDATE sessions SET expires_at=? WHERE id=?`, time.Now().Add(-time.Hour).Unix(), userSession.ID.String()))
	require.ErrorIs(t, s.UpdateAccountAccess(ctx, userSession.ID, root.ID, models.UserRoleAdmin, "active"), db.ErrConflict)
}

// TestActiveGuardUpgrade160 verifies the v5 upgrade, backup and restart without
// rewriting previous migration SQL or relying on the service's count check.
func TestActiveGuardUpgrade160(t *testing.T) {
	ctx := context.Background()
	s, p, _ := migrationDB(t, 5)
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	pending, err := s.CreateListeningProfile(ctx, "Pending", "pending", "", "key")
	require.NoError(t, err)
	require.NoError(t, s.Exec(`UPDATE users SET role='admin' WHERE id=?`, pending.UserID))
	_, err = p.Up(ctx)
	require.NoError(t, err)
	require.Error(t, s.Exec(`UPDATE users SET role='user' WHERE id=?`, root.ID))
	require.Error(t, s.Exec(`UPDATE users SET status='pending',password=X'' WHERE id=?`, root.ID))
	require.Error(t, s.Exec(`DELETE FROM users WHERE id=?`, root.ID))
	require.NoError(t, checkDatabase(s.db))
}

// TestAuthorizationBackup160 validates a v5 snapshot and migration restart on a restored copy.
func TestAuthorizationBackup160(t *testing.T) {
	ctx := context.Background()
	s, _, source := migrationDB(t, 5)
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err = bootstrap(dir, true)
	require.NoError(t, err)
	copies, err := filepath.Glob(target + ".pre-authorization-*.bak")
	require.NoError(t, err)
	require.Len(t, copies, 1)
	saved, err := openExisting(copies[0])
	require.NoError(t, err)
	version, err := schemaVersion(saved)
	require.NoError(t, err)
	require.Equal(t, 5, version)
	require.NoError(t, checkDatabase(saved))
	require.NoError(t, saved.Close())
	restored := t.TempDir()
	require.NoError(t, copyDatabase(copies[0], filepath.Join(restored, "tunetrail.db")))
	restoredPath, err := bootstrap(restored, true)
	require.NoError(t, err)
	conn, err := openExisting(restoredPath)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, checkDatabase(conn))
	var n int
	require.NoError(t, conn.QueryRow(`SELECT COUNT(*) FROM users WHERE id=? AND role='admin' AND status='active'`, root.ID).Scan(&n))
	require.Equal(t, 1, n)
	_, err = bootstrap(restored, true)
	require.NoError(t, err)
}
