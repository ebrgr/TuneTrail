package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBootstrapFresh verifies atomic creation and idempotent restarts.
func TestBootstrapFresh(t *testing.T) {
	dir := t.TempDir()
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "tunetrail.db"), path)
	require.NoFileExists(t, filepath.Join(dir, "koito.db"))
	db, err := openExisting(path)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(db))
	require.NoError(t, db.Close())
	again, err := bootstrap(dir, true)
	require.NoError(t, err)
	require.Equal(t, path, again)
}

// TestBootstrapLegacy preserves v2 and v3 databases, including committed WAL.
func TestBootstrapLegacy(t *testing.T) {
	for _, version := range []int64{2, 3} {
		t.Run(string(rune('0'+version)), func(t *testing.T) {
			store, _, source := migrationDB(t, version)
			dir := filepath.Dir(source)
			_, err := store.db.Exec(`PRAGMA journal_mode=WAL`)
			require.NoError(t, err)
			_, err = store.db.Exec(`CREATE TABLE wal_fixture (value TEXT); INSERT INTO wal_fixture VALUES ('committed')`)
			require.NoError(t, err)
			path, err := bootstrap(dir, true)
			require.NoError(t, err)
			require.FileExists(t, source)
			backup, err := filepath.Glob(filepath.Join(dir, "koito.db.pre-tunetrail-*.bak"))
			require.NoError(t, err)
			require.Len(t, backup, 1)
			// The retained backup must be independently readable and include the WAL row.
			restored, err := openExisting(backup[0])
			require.NoError(t, err)
			var backupValue string
			require.NoError(t, restored.QueryRow(`SELECT value FROM wal_fixture`).Scan(&backupValue))
			require.Equal(t, "committed", backupValue)
			require.NoError(t, checkIntegrity(restored))
			require.NoError(t, restored.Close())
			info, err := os.Stat(backup[0])
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0600), info.Mode().Perm())
			db, err := openExisting(path)
			require.NoError(t, err)
			defer db.Close()
			var got string
			require.NoError(t, db.QueryRow(`SELECT value FROM wal_fixture`).Scan(&got))
			require.Equal(t, "committed", got)
			require.NoError(t, checkDatabase(db))
			again, err := bootstrap(dir, true)
			require.NoError(t, err)
			require.Equal(t, path, again)
			require.NoError(t, store.db.Close())
		})
	}
}

// TestBootstrapConflicts ensures independent or damaged databases are not replaced.
func TestBootstrapConflicts(t *testing.T) {
	dir := t.TempDir()
	_, err := bootstrap(dir, false)
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "koito.db"), []byte("unrelated"), 0600))
	_, err = bootstrap(dir, true)
	require.ErrorContains(t, err, "conflicting databases")
	require.FileExists(t, path)
}

// TestBootstrapConcurrent verifies simultaneous starts publish only one valid target.
func TestBootstrapConcurrent(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	results := make(chan error, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := bootstrap(dir, true); results <- err }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
}

// TestBootstrapCorruptLegacy checks failures do not create a target database.
func TestBootstrapCorruptLegacy(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "koito.db"), []byte("invalid"), 0600))
	_, err := bootstrap(dir, true)
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
	require.False(t, strings.Contains(err.Error(), "password"))
}

// TestBootstrapPathEscaping preserves database paths containing URI delimiters.
func TestBootstrapPathEscaping(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data ?#%")
	require.NoError(t, os.Mkdir(dir, 0700))
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	db, err := openExisting(path)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(db))
	require.NoError(t, db.Close())
}

// TestBootstrapSymlink prevents redirecting the legacy database outside the directory.
func TestBootstrapSymlink(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing"), filepath.Join(dir, "koito.db")))
	_, err := bootstrap(dir, true)
	require.ErrorContains(t, err, "unsafe database path")
	require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
}

// TestBootstrapChangedLegacy prevents silently choosing diverging databases.
func TestBootstrapChangedLegacy(t *testing.T) {
	store, _, source := migrationDB(t, 3)
	require.NoError(t, store.db.Close())
	dir := filepath.Dir(source)
	target, err := bootstrap(dir, true)
	require.NoError(t, err)
	legacy, err := openExisting(source)
	require.NoError(t, err)
	_, err = legacy.Exec(`CREATE TABLE changed_after_migration (value INTEGER)`)
	require.NoError(t, err)
	require.NoError(t, legacy.Close())
	_, err = bootstrap(dir, true)
	require.ErrorContains(t, err, "legacy database changed")
	require.FileExists(t, target)
}

// TestBootstrapPartialFilesNeverBecomeThePublishedDatabase verifies crash leftovers.
func TestBootstrapPartialFilesNeverBecomeThePublishedDatabase(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".tunetrail-migrate-partial.db"), []byte("incomplete"), 0600))
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	db, err := openExisting(path)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(db))
	require.NoError(t, db.Close())
}

// TestBootstrapOrphanedSidecars refuses recovery files without their database.
func TestBootstrapOrphanedSidecars(t *testing.T) {
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		t.Run(suffix, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "tunetrail.db"+suffix), nil, 0600))
			_, err := bootstrap(dir, true)
			require.ErrorContains(t, err, "orphaned database sidecar")
			require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
		})
	}
}

// TestBootstrapExistingV2 verifies backup and migration at the new path.
func TestBootstrapExistingV2(t *testing.T) {
	store, _, source := migrationDB(t, 2)
	require.NoError(t, store.db.Close())
	target := filepath.Join(filepath.Dir(source), "tunetrail.db")
	require.NoError(t, os.Rename(source, target))
	got, err := bootstrap(filepath.Dir(target), true)
	require.NoError(t, err)
	require.Equal(t, target, got)
	backups, err := filepath.Glob(target + ".pre-profiles-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	db, err := openExisting(target)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(db))
	require.NoError(t, db.Close())
}

// TestBootstrapRejectsIncompleteV3 prevents accepting a broken security layout.
func TestBootstrapRejectsIncompleteV3(t *testing.T) {
	store, _, source := migrationDB(t, 3)
	_, err := store.db.Exec(`DROP TRIGGER trg_api_key_profile_immutable`)
	require.NoError(t, err)
	require.NoError(t, store.db.Close())
	_, err = bootstrap(filepath.Dir(source), true)
	require.ErrorContains(t, err, "required trigger")
	require.NoFileExists(t, filepath.Join(filepath.Dir(source), "tunetrail.db"))
}

// TestBootstrapRelativeDirectory supports the default local development setup.
func TestBootstrapRelativeDirectory(t *testing.T) {
	dir, err := os.MkdirTemp(".", "bootstrap-relative-*")
	require.NoError(t, err)
	defer os.RemoveAll(dir)
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(path))
	db, err := openExisting(path)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(db))
	require.NoError(t, db.Close())
}
