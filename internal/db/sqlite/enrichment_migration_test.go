package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestBootstrapEnrichmentFromV23 checks both the upgraded database and its rollback copy.
func TestBootstrapEnrichmentFromV23(t *testing.T) {
	dir := t.TempDir()
	path, err := bootstrapAtVersion(dir, true, 23)
	require.NoError(t, err)
	db, err := openExisting(path)
	require.NoError(t, err)
	s := &Sqlite{db: db}
	testfixtures.Profile(t, s, 1, "metadata-v23", "Metadata", 100)
	execSQL(t, s, `INSERT INTO recordings(id) VALUES(1);
		INSERT INTO occurrences(id,recording_id,presented_title) VALUES(1,1,'Track');
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,100,'v23-listen')`)
	before := queryDigest(t, db, `SELECT * FROM listens ORDER BY event_id`)
	require.NoError(t, db.Close())

	upgraded, err := bootstrap(dir, false)
	require.NoError(t, err)
	require.Equal(t, path, upgraded)
	db, err = openExisting(upgraded)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, checkDatabase(db))
	version, err := schemaVersion(db)
	require.NoError(t, err)
	require.Equal(t, 24, version)
	require.Equal(t, before, queryDigest(t, db, `SELECT * FROM listens ORDER BY event_id`))
	require.Zero(t, scalar(t, &Sqlite{db: db}, `SELECT count(*) FROM metadata_values`))
	backups, err := filepath.Glob(path + ".pre-ingest-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	defer backup.Close()
	require.NoError(t, checkDatabase(backup))
	version, err = schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 23, version)
	require.Equal(t, before, queryDigest(t, backup, `SELECT * FROM listens ORDER BY event_id`))
}

// TestBootstrapRejectsEnrichmentPreview protects databases with reused or retired version numbers.
func TestBootstrapRejectsEnrichmentPreview(t *testing.T) {
	for _, version := range []int{24, 25, 26, 27, 28} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			dir := t.TempDir()
			path, err := bootstrapAtVersion(dir, true, 23)
			require.NoError(t, err)
			db, err := openExisting(path)
			require.NoError(t, err)
			s := &Sqlite{db: db}
			execSQL(t, s, `INSERT INTO goose_db_version(version_id,is_applied) VALUES(?,1)`, version)
			if version == 24 {
				execSQL(t, s, `CREATE TABLE enrichment_recording_claims(fixture TEXT)`)
			}
			require.NoError(t, db.Close())
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			_, err = bootstrap(dir, false)
			require.ErrorContains(t, err, "ISSUE-174 development preview")
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, before, after)
		})
	}
}
