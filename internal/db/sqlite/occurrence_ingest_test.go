package sqlite

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// seedOccurrenceLinks adds two linked receipts, one of which survives a legacy merge.
func seedOccurrenceLinks(t *testing.T, s *Sqlite) {
	t.Helper()
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,70,43);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,state,identity_status,event_id,occurrence_id,recording_id,duplicate_decision,resolution_version)
		SELECT 'linked-'||l.listened_at,1,'source',1,'test','1','fixture',l.listened_at,'submit',0,'second','start','{}','digest','1',
			'active','accepted','resolved',l.event_id,l.track_id,o.recording_id,'different',1
		FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.track_id=70;
		INSERT INTO ingest_revisions
		SELECT profile_id,receipt_id,1,'fixture','different','["fixture"]','{}','fixture-v1',0
		FROM ingest_events WHERE event_id IS NOT NULL`)
}

// TestOccurrenceUpgrade repairs drift from the old merge and preserves committed WAL data in the backup.
func TestOccurrenceUpgrade(t *testing.T) {
	ctx := context.Background()
	s, _, source := migrationDB(t, 9)
	seedOccurrenceLinks(t, s)
	require.NoError(t, s.MergeTracks(ctx, 70, 1))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='linked-43' AND occurrence_id IS NULL`))

	// Copy the quiescent fixture and its committed WAL while the original connection remains open.
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(source + suffix)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(target+suffix, data, 0600))
	}
	_, err := bootstrap(dir, false)
	require.NoError(t, err)

	backups, err := filepath.Glob(target + ".pre-occurrence-ingest-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	defer backup.Close()
	require.NoError(t, checkDatabase(backup))
	require.Equal(t, 9, scalar(t, &Sqlite{db: backup}, `SELECT max(version_id) FROM goose_db_version`))
	require.Equal(t, 1, scalar(t, &Sqlite{db: backup}, `SELECT count(*) FROM ingest_events
		WHERE receipt_id='linked-43' AND occurrence_id IS NULL AND resolution_version=1`))

	for range 2 {
		conn, err := openExisting(target)
		require.NoError(t, err)
		current := &Sqlite{db: conn}
		require.NoError(t, checkDatabase(conn))
		require.Equal(t, 1, scalar(t, current, `SELECT count(*) FROM ingest_events e JOIN listens l USING(profile_id,event_id)
			JOIN occurrences o ON o.id=l.track_id WHERE receipt_id='linked-43' AND e.occurrence_id=1
			AND e.recording_id=o.recording_id AND e.resolution_version=2 AND e.resolver_variant='v9'`))
		require.Equal(t, 2, scalar(t, current, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='linked-43'`))
		require.Equal(t, 1, scalar(t, current, `SELECT count(*) FROM ingest_revisions WHERE author='migration'
			AND decision='different' AND json_extract(evidence,'$.after.occurrence_id')=1`))
		require.NoError(t, conn.Close())
		_, err = bootstrap(dir, false)
		require.NoError(t, err)
	}
}

// TestOccurrenceUpgradeRollback uses a real goose transaction and fails on either side of the repair.
func TestOccurrenceUpgradeRollback(t *testing.T) {
	for _, marker := range []string{"CREATE TEMP TABLE occurrence_link_repairs AS", "DROP TABLE occurrence_link_repairs;"} {
		t.Run(strings.Fields(marker)[0], func(t *testing.T) {
			ctx := context.Background()
			s, _, _ := migrationDB(t, 9)
			seedOccurrenceLinks(t, s)
			require.NoError(t, s.MergeTracks(ctx, 70, 1))
			files := fstest.MapFS{}
			names, err := fs.Glob(migrations.Files, "*.sql")
			require.NoError(t, err)
			for _, name := range names {
				ddl, err := migrations.Files.ReadFile(name)
				require.NoError(t, err)
				if name == "010_occurrence_ingest.sql" {
					require.Contains(t, string(ddl), marker)
					ddl = []byte(strings.Replace(string(ddl), marker, "SELECT * FROM injected_occurrence_failure;\n"+marker, 1))
				}
				files[name] = &fstest.MapFile{Data: ddl}
			}
			broken, err := goose.NewProvider(goose.DialectSQLite3, s.db, files)
			require.NoError(t, err)
			_, err = broken.Up(ctx)
			require.ErrorContains(t, err, "injected_occurrence_failure")
			require.NoError(t, checkDatabase(s.db))
			require.Equal(t, 9, scalar(t, s, `SELECT max(version_id) FROM goose_db_version`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM event_origins`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE author='migration'`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events
				WHERE receipt_id='linked-43' AND occurrence_id IS NULL AND resolution_version=1`))
		})
	}
}

// TestOccurrenceRepairAssertions reports the invalid decision or remaining drift explicitly.
func TestOccurrenceRepairAssertions(t *testing.T) {
	for _, tc := range []struct{ query, message string }{
		{`UPDATE ingest_events SET duplicate_decision=NULL WHERE receipt_id='linked-43'`, "occurrence_repair_missing_decision"},
		{`CREATE TRIGGER skip_repair BEFORE UPDATE OF occurrence_id ON ingest_events
			WHEN NEW.receipt_id='linked-43' BEGIN SELECT RAISE(IGNORE); END`, "occurrence_repair_remaining_drift"},
	} {
		t.Run(tc.message, func(t *testing.T) {
			ctx := context.Background()
			s, provider, _ := migrationDB(t, 9)
			seedOccurrenceLinks(t, s)
			require.NoError(t, s.MergeTracks(ctx, 70, 1))
			execSQL(t, s, tc.query)
			_, err := provider.Up(ctx)
			require.ErrorContains(t, err, tc.message)
			require.Equal(t, 9, scalar(t, s, `SELECT max(version_id) FROM goose_db_version`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE author='migration'`))
		})
	}
}

// TestOccurrenceFlagRequiresActive keeps the flag attached to its active profile.
func TestOccurrenceFlagRequiresActive(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedIdentityHistory(t, s)
	_, err := s.db.Exec(`INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,0,1)`)
	require.ErrorContains(t, err, "requires active ingestion")
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id) VALUES(1)`)
	_, err = s.db.Exec(`UPDATE profile_ingest_flags SET occurrence_write=1 WHERE profile_id=1`)
	require.ErrorContains(t, err, "Occurrence writing cannot be disabled")
	execSQL(t, s, `UPDATE profile_ingest_flags SET active=1,occurrence_write=1 WHERE profile_id=1`)
	_, err = s.db.Exec(`UPDATE profile_ingest_flags SET profile_id=2 WHERE profile_id=1`)
	require.ErrorContains(t, err, "Occurrence writing cannot be disabled")

	// The activation upsert leaves the converted profile's flag untouched.
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,review) VALUES(1,1,0)
		ON CONFLICT(profile_id) DO UPDATE SET active=excluded.active,review=excluded.review`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags WHERE profile_id=1 AND active=1 AND occurrence_write=1`))
}

// TestOccurrenceLinkRequiresIDs rejects incomplete links on insertion and first acceptance.
func TestOccurrenceLinkRequiresIDs(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedOccurrenceLinks(t, s)
	const insert = `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
		batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
		mode,state,identity_status,event_id,occurrence_id,recording_id,duplicate_decision,resolution_version)
		SELECT ?,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,?,'resolved',CASE WHEN ?='received' THEN NULL ELSE event_id END,?,?,duplicate_decision,1
		FROM ingest_events WHERE receipt_id='linked-43'`
	execSQL(t, s, insert, "candidate", "received", "received", nil, nil)
	var recording int
	require.NoError(t, s.db.QueryRow(`SELECT recording_id FROM occurrences WHERE id=70`).Scan(&recording))
	for _, state := range []string{"accepted", "duplicate"} {
		for _, ids := range [][2]any{{nil, recording}, {70, nil}} {
			_, err := s.db.Exec(insert, "incomplete", state, state, ids[0], ids[1])
			require.ErrorContains(t, err, "Inconsistent resolved ingest link")
			_, err = s.db.Exec(`UPDATE ingest_events SET state=?,resolution_version=2,
				event_id=(SELECT event_id FROM ingest_events WHERE receipt_id='linked-43'),occurrence_id=?,recording_id=?
				WHERE receipt_id='candidate'`, state, ids[0], ids[1])
			require.ErrorContains(t, err, "Inconsistent resolved ingest link")
		}
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='candidate' AND state='received'`))
}

// TestOccurrenceGuardCascades covers the real foreign-key order for every legacy deletion root.
func TestOccurrenceGuardCascades(t *testing.T) {
	for _, deletion := range []struct{ name, sql string }{
		{"track", "DELETE FROM tracks WHERE id=70"},
		{"release", "DELETE FROM releases WHERE id=1"},
		{"profile", "DELETE FROM listening_profiles WHERE id=1; DELETE FROM users WHERE id=1"},
	} {
		t.Run(deletion.name, func(t *testing.T) {
			s, _, _ := migrationDB(t, 10)
			seedOccurrenceLinks(t, s)
			execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)`)
			for _, query := range []string{
				"UPDATE profile_ingest_flags SET active=0",
				"UPDATE profile_ingest_flags SET occurrence_write=0",
				"DELETE FROM profile_ingest_flags",
			} {
				_, err := s.db.Exec(query)
				require.ErrorContains(t, err, "Occurrence writing cannot be disabled")
			}
			execSQL(t, s, deletion.sql)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id LIKE 'linked-%'`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_revisions`))
			if deletion.name == "profile" {
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags`))
			}
			require.NoError(t, checkDatabase(s.db))
		})
	}
}

// TestOccurrenceMergeSync keeps the surviving link aligned and deletes the colliding one.
func TestOccurrenceMergeSync(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedOccurrenceLinks(t, s)
	require.NoError(t, s.MergeTracks(context.Background(), 70, 1))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='linked-42'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN occurrences o ON o.id=e.occurrence_id
		WHERE e.receipt_id='linked-43' AND o.id=1 AND e.recording_id=o.recording_id AND e.resolution_version=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions
		WHERE receipt_id='linked-43' AND resolution_version=2 AND reason_codes='["legacy_listen_moved"]'`))
	require.NoError(t, checkDatabase(s.db))
}

// TestOccurrenceMergeRequiresRevision catches an insert skipped under UPDATE OR IGNORE.
func TestOccurrenceMergeRequiresRevision(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedOccurrenceLinks(t, s)
	// A NULL decision is legal on the ledger but violates the revision's NOT NULL.
	execSQL(t, s, `UPDATE ingest_events SET duplicate_decision=NULL,resolution_version=2 WHERE receipt_id='linked-43'`)
	require.ErrorContains(t, s.MergeTracks(context.Background(), 70, 1), "Missing link revision")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE track_id=70 AND listened_at=43`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events
		WHERE receipt_id='linked-43' AND occurrence_id=70 AND resolution_version=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='linked-43'`))
}

// TestOccurrenceGuards exercises registered origins, immutable fields and protected parents.
func TestOccurrenceGuards(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedOccurrenceLinks(t, s)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,70,44,'listenbrainz_api');
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,
			state,identity_status,duplicate_decision,event_id,occurrence_id,recording_id,resolution_version)
		SELECT 'occurrence',1,'source',1,'test','1','fixture',44,'submit',0,'second','start','{}','digest','1','active','occurrence',
			'accepted','resolved','different',l.event_id,l.track_id,o.recording_id,1
		FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.listened_at=44`)
	for _, tc := range []struct{ query, message string }{
		{`UPDATE OR IGNORE listens SET track_id=1 WHERE listened_at=44`, "Cannot move"},
		{`UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=1) WHERE id=70`, "Cannot reassign"},
		{`UPDATE listens SET origin='legacy' WHERE listened_at=44`, "origin is immutable"},
		{`UPDATE ingest_events SET resolver_variant='v9' WHERE receipt_id='occurrence'`, "variant is immutable"},
		{`UPDATE ingest_events SET occurrence_id=1,resolution_version=2 WHERE receipt_id='occurrence'`, "Inconsistent resolved"},
		{`UPDATE ingest_events SET resolution_version=0 WHERE receipt_id='occurrence'`, "newer resolution version"},
		{`INSERT INTO occurrences(recording_id,release_id) SELECT recording_id,1 FROM occurrences WHERE id=1`, "matching track"},
		{`INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,1,99,'bad')`, "Unregistered listen origin"},
	} {
		_, err := s.db.Exec(tc.query)
		require.ErrorContains(t, err, tc.message, tc.query)
	}
	require.ErrorContains(t, s.MergeTracks(context.Background(), 70, 1), "Cannot move")
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens WHERE track_id=70`))
	execSQL(t, s, `INSERT INTO occurrences(id,recording_id,presented_title) SELECT 90,recording_id,'Observed' FROM occurrences WHERE id=1;
		INSERT INTO occurrence_credits VALUES(90,0,'Artist',1);
		DELETE FROM artists WHERE id=1`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM occurrence_credits WHERE name='Artist' AND artist_id IS NULL`))
	execSQL(t, s, `DELETE FROM occurrences WHERE id=90; DELETE FROM tracks WHERE id=70`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM occurrence_credits`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='occurrence'`))
	require.NoError(t, checkDatabase(s.db))
}

// TestOccurrenceStandaloneGuard requires a title and prevents changing the bridge.
func TestOccurrenceStandaloneGuard(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedIdentityHistory(t, s)
	for _, query := range []string{
		`INSERT INTO occurrences(recording_id) SELECT recording_id FROM occurrences WHERE id=1`,
		`INSERT INTO occurrences(recording_id,presented_title) SELECT recording_id,'   ' FROM occurrences WHERE id=1`,
	} {
		_, err := s.db.Exec(query)
		require.ErrorContains(t, err, "title without a release")
	}
	execSQL(t, s, `INSERT INTO occurrences(id,recording_id,presented_title) SELECT 90,recording_id,'Observed' FROM occurrences WHERE id=1;
		INSERT INTO releases(id) VALUES(2);
		UPDATE tracks SET release_id=2 WHERE id=1`)
	require.Equal(t, 2, scalar(t, s, `SELECT release_id FROM occurrences WHERE id=1`))
	for _, query := range []string{
		`UPDATE occurrences SET legacy_track_id=NULL WHERE id=1`,
		`UPDATE occurrences SET release_id=1 WHERE id=90`,
	} {
		_, err := s.db.Exec(query)
		require.ErrorContains(t, err, "Cannot change occurrence bridge")
	}
}

// TestOccurrenceBridgeMBID covers legacy fills and no-op updates at the SQL boundary.
func TestOccurrenceBridgeMBID(t *testing.T) {
	s, _, _ := migrationDB(t, 10)
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO tracks(id,release_id) VALUES(71,1),(72,1);
		UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=1) WHERE id=70;
		UPDATE recordings SET mbid_source='validated' WHERE id=(SELECT recording_id FROM occurrences WHERE id=71)`)
	for _, query := range []string{
		`UPDATE tracks SET musicbrainz_id='fill-secondary' WHERE id=(SELECT id FROM tracks_with_title WHERE title='Intro' AND id=70)`,
		`UPDATE tracks SET musicbrainz_id='fill-validated' WHERE id=71`,
	} {
		_, err := s.db.Exec(query)
		require.ErrorContains(t, err, "shared or validated recording")
	}
	execSQL(t, s, `UPDATE tracks SET musicbrainz_id='fill-legacy' WHERE id=72;
		UPDATE tracks SET musicbrainz_id=musicbrainz_id WHERE id IN (70,71,72)`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recordings WHERE musicbrainz_id='fill-legacy' AND mbid_source='legacy_declared'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recordings WHERE mbid_source='validated' AND musicbrainz_id IS NULL`))
}
