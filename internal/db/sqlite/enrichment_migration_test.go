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
	require.Equal(t, currentSchemaVersion, version)
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

// TestBootstrapEnrichmentRegistryFromV24 upgrades a released v0.75/v0.76 database (ISSUE-214).
func TestBootstrapEnrichmentRegistryFromV24(t *testing.T) {
	for _, configured := range []bool{true, false} {
		t.Run(fmt.Sprint("subsonic configured ", configured), func(t *testing.T) {
			dir := t.TempDir()
			path, err := bootstrapAtVersion(dir, true, 24)
			require.NoError(t, err)
			db, err := openExisting(path)
			require.NoError(t, err)
			s := &Sqlite{db: db}
			execSQL(t, s, `UPDATE enrichment_configuration SET enabled=?`, configured)
			require.NoError(t, db.Close())

			upgraded, err := bootstrap(dir, false)
			require.NoError(t, err)
			db, err = openExisting(upgraded)
			require.NoError(t, err)
			defer db.Close()
			s = &Sqlite{db: db}
			require.NoError(t, checkDatabase(db))
			version, err := schemaVersion(db)
			require.NoError(t, err)
			require.Equal(t, currentSchemaVersion, version)

			// The old switch became the availability of the Subsonic provider, and the table is gone.
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_providers WHERE id='navidrome-subsonic'
				AND enabled=1 AND priority=100 AND identity=1 AND metadata=0 AND eligibility='configured' AND available=?`, configured))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_providers WHERE id='listenbrainz'
				AND enabled=1 AND available=1 AND priority=200 AND eligibility='profile-relay'`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM sqlite_schema WHERE name='enrichment_configuration'`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM sqlite_schema WHERE type='trigger' AND name='trg_enrichment_enqueue'`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_provider_runs`))
			backups, err := filepath.Glob(path + ".pre-ingest-*.bak")
			require.NoError(t, err)
			require.NotEmpty(t, backups, "the upgrade keeps a verified copy of the v24 database")
		})
	}
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

// TestBootstrapDescriptiveMetadataFromV25 preserves the existing event-scoped declarations.
func TestBootstrapDescriptiveMetadataFromV25(t *testing.T) {
	dir := t.TempDir()
	path, err := bootstrapAtVersion(dir, true, 25)
	require.NoError(t, err)
	db, err := openExisting(path)
	require.NoError(t, err)
	s := &Sqlite{db: db}
	testfixtures.Profile(t, s, 1, "metadata-v25", "Metadata", 100)
	execSQL(t, s, `INSERT INTO recordings(id) VALUES(1);
 INSERT INTO occurrences(id,recording_id,presented_title) VALUES(1,1,'Track');
 INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,100,'v25-listen');
 INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,kind,value,instance,source,observed_at)
 VALUES(1,1,1,'v25-listen','mbid','example','instance','fixture','2026-01-01')`)
	before := queryDigest(t, db, `SELECT * FROM metadata_values`)
	require.NoError(t, db.Close())
	_, err = bootstrap(dir, false)
	require.NoError(t, err)
	db, err = openExisting(path)
	require.NoError(t, err)
	defer db.Close()
	s = &Sqlite{db: db}
	require.Equal(t, before, queryDigest(t, db, `SELECT * FROM metadata_values`))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM sqlite_schema WHERE type='trigger' AND name IN ('trg_metadata_scope_insert','trg_metadata_scope_update','trg_metadata_recording_reconcile')`))
	require.NoError(t, checkDatabase(db))
	assertForeignKeys(t, s)
}
