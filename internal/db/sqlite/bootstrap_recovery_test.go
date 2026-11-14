package sqlite

// Recovery tests for the koito.db → tunetrail.db bootstrap (ISSUE-064).
//
// Two failure models are exercised at every phase boundary marked by
// bootstrapPhase:
//   - an injected error (deferred cleanup runs), such as a full disk;
//   - an abrupt process exit (no cleanup, the OS releases the lock), which is
//     what a crash, kill -9 or power loss leaves behind.
// After either, a normal start must publish a complete database with every
// legacy row, keep koito.db unchanged and never publish a partial file.

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

// legacyPhases are the boundaries of a koito.db migration, in execution order.
var legacyPhases = []string{"locked", "backup-written", "backup-synced", "temp-copied", "temp-migrated", "marker-written", "temp-synced", "published"}

const crashExitCode = 3

// legacyFixture creates a v2 koito.db with catalog, account, key and listens.
// The connection stays open, so the rows remain committed only in the WAL.
func legacyFixture(t *testing.T) (dir string, before map[string]tableDigest) {
	t.Helper()
	store, _, source := migrationDB(t, 2)
	_, err := store.db.Exec(`
		INSERT INTO artists(id) VALUES(1);
		INSERT INTO artist_aliases VALUES(1,'Legacy artist','validation',1);
		INSERT INTO releases(id) VALUES(1);
		INSERT INTO release_aliases VALUES(1,'Legacy album','validation',1);
		INSERT INTO artist_releases VALUES(1,1,1);
		INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180);
		INSERT INTO track_aliases VALUES(1,'Legacy track','validation',1);
		INSERT INTO artist_tracks VALUES(1,1,1);
		INSERT INTO users(id,username,role,password) VALUES(1,'validation-admin','admin',x'00');
		INSERT INTO api_keys(key,user_id,created_at,label) VALUES('legacy-token',1,1700000000,'Legacy player');
		CREATE TABLE wal_fixture (value TEXT);
		INSERT INTO wal_fixture VALUES ('committed');`)
	require.NoError(t, err)
	for i := 0; i < 200; i++ {
		_, err = store.db.Exec(`INSERT INTO listens(track_id,listened_at,user_id,client) VALUES(1,?,1,'legacy-player')`, 1700000000+i)
		require.NoError(t, err)
	}
	wal, err := os.Stat(source + "-wal")
	require.NoError(t, err)
	require.Positive(t, wal.Size(), "fixture rows must still be pending in the WAL")
	return filepath.Dir(source), legacyDigests(t, store.db)
}

// tableDigest identifies the legacy columns of a table and a hash of their rows.
type tableDigest struct {
	columns []string
	rows    int
	hash    string
}

func legacyDigests(t *testing.T, db *sql.DB) map[string]tableDigest {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name <> 'goose_db_version'`)
	require.NoError(t, err)
	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	require.NoError(t, rows.Close())
	out := map[string]tableDigest{}
	for _, table := range tables {
		cols, err := db.Query(fmt.Sprintf(`SELECT name FROM pragma_table_info(%q)`, table))
		require.NoError(t, err)
		var columns []string
		for cols.Next() {
			var c string
			require.NoError(t, cols.Scan(&c))
			columns = append(columns, c)
		}
		require.NoError(t, cols.Close())
		out[table] = digestTable(t, db, table, columns)
	}
	return out
}

// digestTable hashes the given columns in a stable order, independent of rowid.
func digestTable(t *testing.T, db *sql.DB, table string, columns []string) tableDigest {
	t.Helper()
	quoted := make([]string, len(columns))
	for i, c := range columns {
		quoted[i] = fmt.Sprintf("quote(%q)", c)
	}
	expr := strings.Join(quoted, "||'|'||")
	rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %q`, expr, table))
	require.NoError(t, err)
	var values []string
	for rows.Next() {
		var v string
		require.NoError(t, rows.Scan(&v))
		values = append(values, v)
	}
	require.NoError(t, rows.Close())
	sort.Strings(values)
	sum := sha256.Sum256([]byte(strings.Join(values, "\n")))
	return tableDigest{columns: columns, rows: len(values), hash: hex.EncodeToString(sum[:])}
}

// requireRecovered checks the published database against the legacy content.
func requireRecovered(t *testing.T, dir string, before map[string]tableDigest, legacyDigest string) {
	t.Helper()
	path, err := bootstrap(dir, true)
	require.NoError(t, err, "a normal restart must complete the migration")
	db, err := openExisting(path)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, checkDatabase(db))
	version, err := schemaVersion(db)
	require.NoError(t, err)
	require.Equal(t, 3, version)
	for table, want := range before {
		got := digestTable(t, db, table, want.columns)
		require.Equal(t, want.rows, got.rows, "rows of %s", table)
		require.Equal(t, want.hash, got.hash, "legacy columns of %s", table)
	}
	require.Equal(t, 200, before["listens"].rows)
	// The marker names the backup actually used; it must exist and be valid.
	var backup string
	require.NoError(t, db.QueryRow(`SELECT backup_name FROM tunetrail_migration WHERE id=1`).Scan(&backup))
	saved, err := openExisting(filepath.Join(dir, backup))
	require.NoError(t, err)
	require.NoError(t, checkIntegrity(saved))
	require.NoError(t, saved.Close())
	// koito.db is never modified by the migration.
	now, err := legacyDigestOf(dir)
	require.NoError(t, err)
	require.Equal(t, legacyDigest, now)
}

func legacyDigestOf(dir string) (string, error) { return legacyDigest(filepath.Join(dir, "koito.db")) }

// leftovers lists files a crash may leave behind (never the published database).
func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	for _, pattern := range []string{".tunetrail-migrate-*.db*", "koito.db.pre-tunetrail-*.bak"} {
		m, err := filepath.Glob(filepath.Join(dir, pattern))
		require.NoError(t, err)
		for _, p := range m {
			out = append(out, filepath.Base(p))
		}
	}
	return out
}

// TestBootstrapRecoversFromInjectedFailureAtEveryPhase simulates a full disk
// (ENOSPC) at each boundary: startup fails, nothing partial is published and a
// later start completes the migration.
func TestBootstrapRecoversFromInjectedFailureAtEveryPhase(t *testing.T) {
	for _, phase := range legacyPhases {
		t.Run(phase, func(t *testing.T) {
			dir, before := legacyFixture(t)
			legacy, err := legacyDigestOf(dir)
			require.NoError(t, err)
			injected := fmt.Errorf("injected at %s: %w", phase, syscall.ENOSPC)
			t.Cleanup(func() { bootstrapPhase = func(string) error { return nil } })
			bootstrapPhase = func(p string) error {
				if p == phase {
					return injected
				}
				return nil
			}
			_, err = bootstrap(dir, true)
			require.ErrorIs(t, err, syscall.ENOSPC)
			if phase == "published" {
				// The link already happened; the database is complete and is reused.
				require.FileExists(t, filepath.Join(dir, "tunetrail.db"))
			} else {
				require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
			}
			// Deferred cleanup removes the temporary database on an error return.
			tmp, err := filepath.Glob(filepath.Join(dir, ".tunetrail-migrate-*.db"))
			require.NoError(t, err)
			require.Empty(t, tmp)
			bootstrapPhase = func(string) error { return nil }
			requireRecovered(t, dir, before, legacy)
		})
	}
}

// TestBootstrapRecoversFromCrashAtEveryPhase terminates a child process at each
// boundary, leaving whatever an abrupt exit leaves, and then restarts.
func TestBootstrapRecoversFromCrashAtEveryPhase(t *testing.T) {
	for _, phase := range legacyPhases {
		t.Run(phase, func(t *testing.T) {
			dir, before := legacyFixture(t)
			legacy, err := legacyDigestOf(dir)
			require.NoError(t, err)
			crashBootstrap(t, dir, phase)
			if phase == "published" {
				require.FileExists(t, filepath.Join(dir, "tunetrail.db"))
			} else {
				require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
			}
			left := leftovers(t, dir)
			requireRecovered(t, dir, before, legacy)
			// Leftovers are never adopted; the restart keeps them for manual cleanup.
			t.Logf("crash at %s left %d file(s): %v", phase, len(left), left)
		})
	}
}

// crashBootstrap runs bootstrap in a child test process that exits at phase.
func crashBootstrap(t *testing.T, dir, phase string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestBootstrapCrashHelper$")
	cmd.Env = append(os.Environ(), "TUNETRAIL_CRASH_DIR="+dir, "TUNETRAIL_CRASH_AT="+phase)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	require.True(t, errors.As(err, &exit), "child must exit abruptly at %s; output: %s", phase, out)
	require.Equal(t, crashExitCode, exit.ExitCode(), "child output: %s", out)
}

// TestBootstrapCrashHelper is the child side of crashBootstrap; it is skipped
// unless the parent test sets the crash environment.
func TestBootstrapCrashHelper(t *testing.T) {
	dir, phase := os.Getenv("TUNETRAIL_CRASH_DIR"), os.Getenv("TUNETRAIL_CRASH_AT")
	if dir == "" || phase == "" {
		t.Skip("helper process only")
	}
	bootstrapPhase = func(p string) error {
		if p == phase {
			os.Exit(crashExitCode) // no deferred cleanup, like a killed process
		}
		return nil
	}
	_, err := bootstrap(dir, true)
	t.Fatalf("phase %s was never reached: %v", phase, err)
}

// TestBootstrapInPlaceV2RecoversAtEveryPhase covers an existing tunetrail.db
// still at schema v2, migrated in place after its own backup.
func TestBootstrapInPlaceV2RecoversAtEveryPhase(t *testing.T) {
	for _, phase := range []string{"inplace-backup-synced", "inplace-migrated"} {
		t.Run(phase, func(t *testing.T) {
			dir, before := legacyFixture(t)
			// Present the v2 database under the new name, as an older build would.
			require.NoError(t, copyDatabase(filepath.Join(dir, "koito.db"), filepath.Join(dir, "tunetrail.db")))
			require.NoError(t, os.Remove(filepath.Join(dir, "koito.db")))
			crashBootstrap(t, dir, phase)
			path, err := bootstrap(dir, true)
			require.NoError(t, err)
			db, err := openExisting(path)
			require.NoError(t, err)
			defer db.Close()
			version, err := schemaVersion(db)
			require.NoError(t, err)
			require.Equal(t, 3, version)
			for table, want := range before {
				got := digestTable(t, db, table, want.columns)
				require.Equal(t, want.hash, got.hash, "legacy columns of %s", table)
			}
		})
	}
}

// copyDatabase takes a consistent copy, including committed WAL content.
func copyDatabase(src, dst string) error {
	db, err := sql.Open("sqlite", sqliteURI(src)+"?mode=ro")
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`VACUUM INTO ?`, dst)
	return err
}

// TestBootstrapRejectsUnknownSchemaVersions covers the version guards: a legacy
// or current database from a newer or unknown schema is refused untouched.
func TestBootstrapRejectsUnknownSchemaVersions(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		dir, _ := legacyFixture(t)
		db, err := sql.Open("sqlite", filepath.Join(dir, "koito.db"))
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO goose_db_version(version_id,is_applied) VALUES(4,1)`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
		legacy, err := legacyDigestOf(dir)
		require.NoError(t, err)
		_, err = bootstrap(dir, true)
		require.ErrorContains(t, err, "unsupported")
		require.NoFileExists(t, filepath.Join(dir, "tunetrail.db"))
		require.Empty(t, leftovers(t, dir), "no backup or temporary file for a refused version")
		now, err := legacyDigestOf(dir)
		require.NoError(t, err)
		require.Equal(t, legacy, now)
	})
	t.Run("current", func(t *testing.T) {
		dir := t.TempDir()
		path, err := bootstrap(dir, true)
		require.NoError(t, err)
		db, err := sql.Open("sqlite", path)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO goose_db_version(version_id,is_applied) VALUES(99,1)`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
		before, err := legacyDigest(path)
		require.NoError(t, err)
		_, err = bootstrap(dir, true)
		require.ErrorContains(t, err, "unsupported schema version 99")
		after, err := legacyDigest(path)
		require.NoError(t, err)
		require.Equal(t, before, after, "a refused database is not modified")
	})
}

// TestBootstrapRestoreFromBackup rehearses the documented recovery: place the
// retained .bak as koito.db in an empty directory and start again.
func TestBootstrapRestoreFromBackup(t *testing.T) {
	dir, before := legacyFixture(t)
	path, err := bootstrap(dir, true)
	require.NoError(t, err)
	db, err := openExisting(path)
	require.NoError(t, err)
	var backup string
	require.NoError(t, db.QueryRow(`SELECT backup_name FROM tunetrail_migration WHERE id=1`).Scan(&backup))
	require.NoError(t, db.Close())
	restore := t.TempDir()
	raw, err := os.ReadFile(filepath.Join(dir, backup))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(restore, "koito.db"), raw, 0600))
	legacy, err := legacyDigestOf(restore)
	require.NoError(t, err)
	requireRecovered(t, restore, before, legacy)
}

// TestBootstrapReconcileAfterLegacyChange follows step 5 of the recovery
// procedure in DEPLOYMENT.md: after koito.db changes post-migration, startup is
// refused, and either copy can be chosen by moving the other one away.
func TestBootstrapReconcileAfterLegacyChange(t *testing.T) {
	changed := func(t *testing.T) string {
		dir, _ := legacyFixture(t)
		_, err := bootstrap(dir, true)
		require.NoError(t, err)
		// An older release writes to koito.db after the migration.
		db, err := sql.Open("sqlite", filepath.Join(dir, "koito.db"))
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO wal_fixture VALUES ('written by an older release')`)
		require.NoError(t, err)
		require.NoError(t, db.Close())
		_, err = bootstrap(dir, true)
		require.ErrorContains(t, err, "legacy database changed after migration")
		return dir
	}
	count := func(t *testing.T, path string) int {
		db, err := openExisting(path)
		require.NoError(t, err)
		defer db.Close()
		var n int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM wal_fixture`).Scan(&n))
		return n
	}
	t.Run("keep tunetrail.db", func(t *testing.T) {
		dir := changed(t)
		aside := t.TempDir()
		for _, name := range []string{"koito.db", "koito.db-wal", "koito.db-shm"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				require.NoError(t, os.Rename(filepath.Join(dir, name), filepath.Join(aside, name)))
			}
		}
		path, err := bootstrap(dir, true)
		require.NoError(t, err)
		require.Equal(t, 1, count(t, path), "the migrated history is kept")
	})
	t.Run("start over from koito.db", func(t *testing.T) {
		dir := changed(t)
		aside := t.TempDir()
		for _, name := range []string{"tunetrail.db", "tunetrail.db-wal", "tunetrail.db-shm"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				require.NoError(t, os.Rename(filepath.Join(dir, name), filepath.Join(aside, name)))
			}
		}
		path, err := bootstrap(dir, true)
		require.NoError(t, err)
		require.Equal(t, 2, count(t, path), "the legacy database, including the newer row, is migrated again")
	})
}
