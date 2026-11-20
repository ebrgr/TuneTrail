package sqlite

import (
	"path/filepath"
	"testing"

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
	_, err := bootstrap(dir, false)
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
	require.Equal(t, 9, version)
	for name, digest := range before {
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
		event_id=(SELECT event_id FROM listens WHERE rowid=10),closed_at=1 WHERE receipt_id='admitted';
		INSERT INTO ingest_revisions VALUES(1,'admitted',1,'fixture','different','["fixture"]','{}','1',1);
		INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,state,closed_at,result)
		SELECT 'o',1,'source',event_id,'listen','key',1,'discarded',1,'{"reason":"pre_082_best_effort"}' FROM listens WHERE rowid=10`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM (`+queue+`)`))
	require.NoError(t, checkIntegrity(conn))
}
