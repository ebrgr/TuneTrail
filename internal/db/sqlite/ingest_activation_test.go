package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestIngestActivationUpgrade verifies the v8 snapshot and the v9 receipt boundary.
func TestIngestActivationUpgrade(t *testing.T) {
	s, _, source := migrationDB(t, 8)
	seedIdentityHistory(t, s)
	execSQL(t, s, `UPDATE ingest_events SET state='pending_duplicate' WHERE receipt_id='shadow'`)
	before := legacyDigests(t, s.db)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrapAtVersion(dir, false, 20)
	require.NoError(t, err)

	backups, err := filepath.Glob(target + ".pre-ingest-activation-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	require.NoError(t, checkDatabase(backup))
	version, err := schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 8, version)
	require.Equal(t, before, legacyDigests(t, backup))
	require.NoError(t, backup.Close())

	conn, err := openExisting(target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	s = &Sqlite{db: conn}
	require.NoError(t, checkDatabase(conn))
	version, err = schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, 20, version)
	for name, digest := range before {
		if name == "event_origins" {
			// Migrations 010 and 014 append origins while keeping the legacy one.
			require.Equal(t, 8, scalar(t, s, `SELECT count(*) FROM event_origins`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM event_origins WHERE origin='legacy'`))
			continue
		}
		require.Equal(t, digest, digestTable(t, conn, name, digest.columns), name)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE mode='shadow'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags`))
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id) VALUES(1)`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags WHERE active=0 AND review=0`))

	for _, tc := range []struct{ query, guard string }{
		{`UPDATE profile_ingest_flags SET review=2`, "CHECK constraint failed: review IN (0,1)"},
		{`UPDATE ingest_events SET mode='active' WHERE receipt_id='shadow'`, "Receipt mode is immutable"},
		{`UPDATE ingest_events SET state='accepted',event_id=(SELECT event_id FROM listens WHERE rowid=10)`, "shadow receipts cannot materialize decisions"},
		{`INSERT INTO ingest_revisions VALUES(1,'shadow',1,'fixture','review','["fixture"]','{}','1',0)`, "Receipt revision requires an active receipt"},
		{`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at)
		  SELECT 'blocked',1,'source',event_id,'listen','blocked',0 FROM listens WHERE rowid=10`, "Outbox intent requires an accepted active receipt"},
	} {
		_, err := conn.Exec(tc.query)
		require.ErrorContains(t, err, tc.guard, tc.query)
	}

	const insert = `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
		batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,state)
		VALUES('admitted',1,'source',1,'test','1','batch',0,'submit',0,'second','start','{}','digest','1',?,'pending_duplicate')`
	_, err = conn.Exec(insert, "unknown")
	require.ErrorContains(t, err, "CHECK constraint failed: mode IN ('shadow','active')")
	execSQL(t, s, insert, "active")
	_, err = conn.Exec(`UPDATE ingest_events SET mode='shadow' WHERE receipt_id='admitted'`)
	require.ErrorContains(t, err, "Receipt mode is immutable")

	const queue = `SELECT receipt_id FROM ingest_events WHERE mode='active' AND closed_at IS NULL AND state='pending_duplicate' ORDER BY next_attempt_at,received_at`
	var receiptID, plan string
	var id, parent, unused int
	require.NoError(t, conn.QueryRow(queue).Scan(&receiptID))
	require.Equal(t, "admitted", receiptID)
	require.NoError(t, conn.QueryRow("EXPLAIN QUERY PLAN "+queue).Scan(&id, &parent, &unused, &plan))
	require.Contains(t, plan, "idx_ingest_active_queue")

	// Resolution and outbox writes succeed with admission still disabled.
	execSQL(t, s, `UPDATE ingest_events SET state='accepted',identity_status='resolved',resolution_version=1,
		event_id=(SELECT event_id FROM listens WHERE rowid=10),occurrence_id=(SELECT track_id FROM listens WHERE rowid=10),
		recording_id=(SELECT o.recording_id FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.rowid=10),
		closed_at=1 WHERE receipt_id='admitted';
		INSERT INTO ingest_revisions VALUES(1,'admitted',1,'fixture','different','["fixture"]','{}','1',1);
		INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,state,closed_at,result)
		SELECT 'o',1,'source',event_id,'listen','key',1,'discarded',1,'{"reason":"pre_082_best_effort"}' FROM listens WHERE rowid=10`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM (`+queue+`)`))
	require.NoError(t, checkIntegrity(conn))
}

// TestIngestDigestUpgrade preserves v1 keyed retries while detecting changes to v1 fields.
func TestIngestDigestUpgrade(t *testing.T) {
	ctx := context.Background()
	s, provider, _ := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	// Store the v1 bytes and batch digest exactly as the old adapter wrote them.
	legacySnapshot := `{"artist":"A","title":"T"}`
	oldDigest := ingest.Digest([]byte(legacySnapshot))
	batchDigest := ingest.Digest([]byte(fmt.Sprintf(`["%s"]`, oldDigest)))
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,client_idempotency_key,batch_digest,mode,state,duplicate_decision,identity_status,closed_at,event_id,occurrence_id,recording_id,resolution_version)
		VALUES('v1',1,'listenbrainz:1',1,'listenbrainz','1','batch',0,'single',0,'estimated','unknown',?,?,'sha256-json-v1','key',?,'active','duplicate','same','resolved',1,(SELECT event_id FROM listens WHERE rowid=10),1,(SELECT recording_id FROM occurrences WHERE id=1),1)`,
		legacySnapshot, oldDigest, batchDigest)
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: "key", ReceivedAt: time.Now(),
		Items: []ingest.Item{{Snapshot: ingest.Snapshot{Artist: "A", Title: "T",
			ArtistMbidMappings: []ingest.ArtistMapping{{Name: "A", MBID: "12345678-1234-4321-abcd-000000000001"}}}}}}
	execSQL(t, s, `DELETE FROM ingest_events WHERE mode='shadow' AND event_id IS NULL`)
	_, err := provider.UpTo(ctx, currentSchemaVersion)
	require.NoError(t, err)
	retry, err := s.AdmitOccurrence(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, "v1", retry[0].ReceiptID)
	require.Equal(t, "same", retry[0].Decision)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id='v1'`))
	batch.Items[0].Snapshot.Title = "Changed"
	retry, err = s.AdmitOccurrence(ctx, batch)
	require.NoError(t, err)
	require.True(t, retry[0].Replay)
	require.Equal(t, "v1", retry[0].ReceiptID)
	require.Equal(t, "same", retry[0].Decision)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id='v1' AND digest_version=?`, ingest.DigestVersion))
}

// TestExpiredIngestLeaseCanBeReclaimed checks CAS while the new claim is still open.
func TestExpiredIngestLeaseCanBeReclaimed(t *testing.T) {
	ctx := context.Background()
	s, provider, path := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE mode='shadow'`)
	_, err := provider.UpTo(ctx, currentSchemaVersion)
	require.NoError(t, err)
	item := ingest.Snapshot{Artist: "Artist", Title: "Lease Track", Release: "Lease Album"}
	id := admitOccurrenceFixture(t, s, item, 100, "lease")
	first, _, _, err := s.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, id, first.id)

	conn, err := openExisting(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	worker := &Sqlite{db: conn}
	unavailable, _, _, err := worker.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Empty(t, unavailable.id)
	execSQL(t, s, `UPDATE ingest_events SET lease_until=unixepoch()-1 WHERE receipt_id=?`, first.id)

	second, _, _, err := worker.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, first.id, second.id)
	require.NotEqual(t, first.lease, second.lease)
	require.Equal(t, first.work+1, second.work)
	first.unix = 100
	second.unix = 100
	first.snapshot, second.snapshot = item, item
	require.NoError(t, s.finishOccurrence(ctx, first, occurrenceProof{}))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=100`))
	require.NoError(t, worker.finishOccurrence(ctx, second, occurrenceProof{}))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=100`))
}
