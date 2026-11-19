package sqlite

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// seedIdentityHistory includes independently attributed rows and a v7 shadow receipt.
func seedIdentityHistory(t *testing.T, s *Sqlite) {
	t.Helper()
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	testfixtures.Profile(t, s, 2, "two", "Two", 0)
	seedMusic(t, s)
	execSQL(t, s, `INSERT INTO tracks(id,release_id,duration) VALUES(70,1,100);
		INSERT INTO track_aliases VALUES (1,'Intro','Canonical',1),(70,'Intro','Canonical',1),(70,'Opening','Import',0);
		INSERT INTO artist_tracks VALUES(1,1,1),(1,70,1);
		INSERT INTO listens(rowid,profile_id,track_id,listened_at,user_id,client)
		VALUES (10,1,1,42,NULL,'client-a'),(20,1,70,42,1,'client-b'),(30,2,1,42,2,'client-c');
		INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0)`)
	insertReceipt(t, s, "shadow")
	execSQL(t, s, `UPDATE ingest_events SET duplicate_decision='review',rule_version='shadow-v2',
		external_id_trusted=1,external_namespace='fixture',external_event_id='stable' WHERE receipt_id='shadow';
		INSERT INTO ingest_event_details(profile_id,receipt_id,resolution_version,kind,payload)
		VALUES(1,'shadow',0,'diagnostic','{}');
		INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
		VALUES('variant',1,'shadow',1,'alternate','1','{}',0)`)
}

// TestRecordingIdentityUpgrade proves both failure atomicity and preservation of v7 data.
func TestRecordingIdentityUpgrade(t *testing.T) {
	ctx := context.Background()
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	before := legacyDigests(t, s.db)
	files := fstest.MapFS{}
	names, err := fs.Glob(migrations.Files, "*.sql")
	require.NoError(t, err)
	for _, name := range names {
		content, err := migrations.Files.ReadFile(name)
		require.NoError(t, err)
		if name == "008_recording_identity.sql" {
			content = []byte(strings.Replace(string(content), "DROP TABLE identity_assertion;",
				"INSERT INTO identity_assertion VALUES ('injected late failure',0);", 1))
		}
		files[name] = &fstest.MapFile{Data: content}
	}
	broken, err := goose.NewProvider(goose.DialectSQLite3, s.db, files)
	require.NoError(t, err)
	_, err = broken.Up(ctx)
	require.ErrorContains(t, err, "CHECK constraint failed")
	require.Equal(t, 7, scalar(t, s, `SELECT max(version_id) FROM goose_db_version WHERE is_applied=1`))
	require.Equal(t, before, legacyDigests(t, s.db))
	require.NoError(t, checkDatabase(s.db))

	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(s.db))
	for name, digest := range before {
		require.Equal(t, digest, digestTable(t, s.db, name, digest.columns), name)
	}
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens WHERE length(event_id)=32 AND origin='legacy'`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=2`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM occurrences WHERE legacy_track_id=id AND id IN (1,70)`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='shadow' AND event_id IS NULL AND duplicate_decision='review'`))
	after := legacyDigests(t, s.db)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, after, legacyDigests(t, s.db))
	_, err = provider.Down(ctx)
	require.Error(t, err)
	require.NoError(t, checkDatabase(s.db))
}

// TestRecordingIdentityBridge exercises the legacy UPSERT, catalog mutations and cutoff guards.
func TestRecordingIdentityBridge(t *testing.T) {
	ctx := context.Background()
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(ctx)
	require.NoError(t, err)
	for _, query := range []string{
		`INSERT INTO occurrences(recording_id,release_id) SELECT id,1 FROM recordings LIMIT 1`,
		`INSERT INTO occurrences(id,recording_id,legacy_track_id) SELECT 71,id,71 FROM recordings LIMIT 1`,
		`UPDATE tracks SET rowid=2 WHERE id=1`,
		`UPDATE listens SET event_id='renamed'`,
		`UPDATE listens SET origin='native'`,
		`INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,1,100,'native')`,
		`INSERT INTO ingest_revisions VALUES(1,'shadow',1,'fixture','review','["fixture"]','{}','1',0)`,
		`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at)
		SELECT 'o',1,'source',event_id,'listen','key',0 FROM listens WHERE rowid=10`,
	} {
		_, err := s.db.Exec(query)
		require.Error(t, err, query)
	}

	opts := db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(42, 0)}
	inserted, err := s.SaveListenWithResult(ctx, opts)
	require.NoError(t, err)
	require.False(t, inserted)
	opts.Time = time.Unix(43, 0)
	inserted, err = s.SaveListenWithResult(ctx, opts)
	require.NoError(t, err)
	require.True(t, inserted)

	execSQL(t, s, `INSERT INTO releases(id) VALUES(2);
		INSERT INTO tracks(id,release_id) VALUES(100,2);
		UPDATE tracks SET musicbrainz_id='32dd4768-7460-4a4e-aee7-e2dfc87299d3' WHERE id=100;
		UPDATE tracks SET release_id=1 WHERE id=100`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM occurrences o JOIN recordings r ON r.id=o.recording_id
		WHERE o.id=100 AND o.release_id=1 AND r.musicbrainz_id IS NOT NULL`))
	require.NoError(t, s.MergeTracks(ctx, 70, 1))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM occurrences WHERE id=70`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND track_id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=1`))
	execSQL(t, s, `DELETE FROM releases WHERE id=1`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM recordings`))
	require.NoError(t, checkDatabase(s.db))
	execSQL(t, s, `CREATE INDEX unexpected_legacy_index ON listens(client)`)
	require.ErrorContains(t, checkDatabase(s.db), "unexpected index unexpected_legacy_index")
}

// TestRecordingIdentityExactDeletion simulates the future schema cut in an isolated fixture.
func TestRecordingIdentityExactDeletion(t *testing.T) {
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(context.Background())
	require.NoError(t, err)
	execSQL(t, s, `DROP INDEX idx_listens_legacy_identity;
		DROP TRIGGER trg_identity_legacy_occurrence;
		DROP TRIGGER trg_ingest_event_inactive;
		DROP TRIGGER trg_ingest_event_inactive_update;
		DROP TRIGGER trg_ingest_revision_inactive;
		DROP TRIGGER trg_outbox_inactive;
		INSERT INTO occurrences(recording_id,release_id) SELECT recording_id,release_id FROM occurrences WHERE id=1;
		INSERT INTO occurrences(recording_id) SELECT recording_id FROM occurrences WHERE id=1;
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,42,'survives')`)
	for _, query := range []string{
		`UPDATE ingest_events SET state='accepted' WHERE receipt_id='shadow'`,
		`UPDATE ingest_events SET state='accepted',event_id=(SELECT event_id FROM listens WHERE rowid=30) WHERE receipt_id='shadow'`,
		`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at)
		SELECT 'wrong',1,'source',event_id,'listen','wrong',0 FROM listens WHERE rowid=30`,
	} {
		_, err := s.db.Exec(query)
		require.Error(t, err, query)
	}
	execSQL(t, s, `UPDATE ingest_events SET state='accepted',event_id=(SELECT event_id FROM listens WHERE rowid=10),
		client_idempotency_key='client-key',batch_digest='batch',
		file_digest=printf('%064d',0),file_format='fixture',format_version='1' WHERE receipt_id='shadow';
		INSERT INTO ingest_revisions VALUES(1,'shadow',1,'fixture','different','["fixture"]','{}','1',0);
		INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at)
		SELECT 'o',1,'source',event_id,'listen','key',0 FROM listens WHERE rowid=10;
		INSERT INTO outbox_details(profile_id,outbox_id,payload) VALUES(1,'o','{}')`)
	var oldID string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE rowid=10`).Scan(&oldID))
	tx, err := s.db.Begin()
	require.NoError(t, err)
	_, err = tx.Exec(`DELETE FROM listens WHERE profile_id=1 AND event_id=?`, oldID)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='shadow'`))
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND event_id=?`, oldID)
	for _, table := range []string{"ingest_events", "ingest_observations", "ingest_revisions", "ingest_event_details", "outbox", "outbox_details"} {
		require.Zero(t, scalar(t, s, "SELECT count(*) FROM "+table), table)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE event_id='survives'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM integrations WHERE id='source'`))

	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,42)`)
	var newID string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE profile_id=1 AND track_id=1 AND listened_at=42 AND event_id<>'survives'`).Scan(&newID))
	require.NotEqual(t, oldID, newID)
	insertReceipt(t, s, "reimport")
	execSQL(t, s, `UPDATE ingest_events SET state='accepted',event_id=?,
		external_id_trusted=1,external_namespace='fixture',external_event_id='stable',
		client_idempotency_key='client-key',batch_digest='batch',
		file_digest=printf('%064d',0),file_format='fixture',format_version='1' WHERE receipt_id='reimport'`, newID)
	execSQL(t, s, `DELETE FROM releases WHERE id=1`)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM occurrences WHERE legacy_track_id IS NULL AND release_id IS NULL`))
	require.NoError(t, checkIntegrity(s.db))
}

// TestRecordingIdentityCursor uses the future ordering over an isolated post-cut fixture.
func TestRecordingIdentityCursor(t *testing.T) {
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(context.Background())
	require.NoError(t, err)
	execSQL(t, s, `DROP INDEX idx_listens_legacy_identity;
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES
			(1,1,41,'a'),(1,1,41,'b'),(2,1,41,'other'),(1,1,41,'c')`)
	var anchor string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE profile_id=1 ORDER BY listened_at,event_id LIMIT 1`).Scan(&anchor))
	require.Equal(t, "a", anchor)
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND event_id=?`, anchor)
	var next string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens
		WHERE profile_id=1 AND (listened_at,event_id)>(41,?)
		ORDER BY listened_at,event_id LIMIT 1`, anchor).Scan(&next))
	require.Equal(t, "b", next)
}

// TestRecordingIdentityBackupRestore restores a v7 snapshot independently of the upgraded file.
func TestRecordingIdentityBackupRestore(t *testing.T) {
	s, _, source := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	original, err := openExisting(target)
	require.NoError(t, err)
	_, err = original.Exec(`PRAGMA journal_mode=WAL; INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(2,70,44,'wal')`)
	require.NoError(t, err)
	defer original.Close()
	before := legacyDigests(t, original)
	_, err = original.Exec(`CREATE INDEX unexpected_legacy_index ON listens(client)`)
	require.NoError(t, err)
	_, err = bootstrap(dir, false)
	require.ErrorContains(t, err, "unexpected index unexpected_legacy_index")
	_, err = original.Exec(`DROP INDEX unexpected_legacy_index`)
	require.NoError(t, err)

	_, err = bootstrap(dir, false)
	require.NoError(t, err)
	backups, err := filepath.Glob(target + ".pre-recording-identity-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	restoredDir := t.TempDir()
	restoredPath := filepath.Join(restoredDir, "tunetrail.db")
	require.NoError(t, copyDatabase(backups[0], restoredPath))
	restored, err := openExisting(restoredPath)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(restored))
	version, err := schemaVersion(restored)
	require.NoError(t, err)
	require.Equal(t, 7, version)
	require.Equal(t, before, legacyDigests(t, restored))
	require.NoError(t, restored.Close())
	_, err = bootstrap(restoredDir, false)
	require.NoError(t, err)
}

// TestRecordingIdentityReferenceOrigins exercises the removable gate and permanent FK.
func TestRecordingIdentityReferenceOrigins(t *testing.T) {
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, `PRAGMA foreign_keys`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM event_origins WHERE origin='legacy'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM event_origins`))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,99)`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE listened_at=99 AND origin='legacy'`))
	_, err = s.db.Exec(`DELETE FROM event_origins WHERE origin='legacy'`)
	require.ErrorContains(t, err, "FOREIGN KEY")
	execSQL(t, s, `DROP TRIGGER trg_identity_legacy_listen_insert`)
	const insert = `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,1,100,'fixture')`
	_, err = s.db.Exec(insert)
	require.ErrorContains(t, err, "FOREIGN KEY")
	execSQL(t, s, `INSERT INTO event_origins VALUES('fixture','Test-only origin')`)
	execSQL(t, s, insert)
	require.NoError(t, checkIntegrity(s.db))
}

// TestRecordingIdentityAllocator verifies that ordinary writes never reuse deleted occurrence IDs.
func TestRecordingIdentityAllocator(t *testing.T) {
	ctx := context.Background()
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(ctx)
	require.NoError(t, err)
	execSQL(t, s, `DELETE FROM tracks WHERE id=70; DROP TRIGGER trg_identity_legacy_occurrence;
		INSERT INTO occurrences(recording_id,release_id) SELECT recording_id,1 FROM occurrences WHERE id=1`)
	independent := scalar(t, s, `SELECT id FROM occurrences WHERE legacy_track_id IS NULL`)
	require.Greater(t, independent, 70)
	_, err = s.db.Exec(`INSERT INTO tracks(id,release_id) VALUES(?,1)`, independent)
	require.ErrorContains(t, err, "Occurrence ID already exists")
	track, err := s.SaveTrack(ctx, db.SaveTrackOpts{AlbumID: 1, ArtistIDs: []int32{1}, Title: "New track"})
	require.NoError(t, err)
	require.Greater(t, int(track.ID), independent)
	require.NoError(t, s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 1, TrackID: track.ID, Time: time.Unix(90, 0)}))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens l JOIN occurrences o ON o.id=l.track_id
		WHERE l.listened_at=90 AND o.legacy_track_id=?`, track.ID))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE track_id IN (70,?)`, independent))
}

// TestRecordingIdentityReceiptGate keeps shadow decisions valid while rejecting resolution.
func TestRecordingIdentityReceiptGate(t *testing.T) {
	s, provider, _ := migrationDB(t, 7)
	seedIdentityHistory(t, s)
	_, err := provider.Up(context.Background())
	require.NoError(t, err)
	execSQL(t, s, `UPDATE ingest_events SET duplicate_decision='same',state='pending_duplicate'`)
	for _, q := range []string{
		`UPDATE ingest_events SET state='retryable'`,
		`UPDATE ingest_events SET identity_status='resolved'`,
		`UPDATE ingest_events SET occurrence_id=1`,
		`UPDATE ingest_events SET recording_id=1`,
		`UPDATE ingest_events SET resolution_version=1`,
		`UPDATE ingest_events SET closed_at=1`,
		`UPDATE ingest_events SET state='accepted',event_id=(SELECT event_id FROM listens WHERE rowid=10)`,
		`INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
			batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state)
			SELECT 'blocked',profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,'retryable'
			FROM ingest_events WHERE receipt_id='shadow'`,
	} {
		_, err := s.db.Exec(q)
		require.ErrorContains(t, err, "Receipt resolution requires pipeline activation", q)
	}
}

// TestRecordingIdentityPreflight protects custom objects on every pre-008 entry path.
func TestRecordingIdentityPreflight(t *testing.T) {
	for version := int64(3); version <= 6; version++ {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			s, _, source := migrationDB(t, version)
			execSQL(t, s, `CREATE INDEX unrelated_artist_image ON artists(image)`)
			require.NoError(t, checkDatabase(s.db))
			execSQL(t, s, `CREATE INDEX unexpected_listens_client ON listens(client)`)
			for _, name := range []string{"tunetrail.db", "koito.db"} {
				dir := t.TempDir()
				require.NoError(t, copyDatabase(source, filepath.Join(dir, name)))
				_, err := bootstrap(dir, false)
				require.ErrorContains(t, err, "unexpected index unexpected_listens_client")
			}
		})
	}
}
