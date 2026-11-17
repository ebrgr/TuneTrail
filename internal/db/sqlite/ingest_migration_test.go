package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ingestFixture creates two unrelated profiles and an inactive receipt.
func ingestFixture(t *testing.T) *Sqlite {
	t.Helper()
	s, err := NewInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.db.Close()) })
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'one','One',0),(2,'two','Two',0)`)
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0),('other',2,'test',0)`)
	insertReceipt(t, s, "receipt")
	return s
}

// insertReceipt keeps every test envelope independent of musical identities.
func insertReceipt(t *testing.T, s *Sqlite, id string) {
	t.Helper()
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version) VALUES(?,1,'source',1,'test','1','batch',0,'submit',0,'second','start','{}','digest','1')`, id)
}

// TestIngestConstraints exercises trusted keys, scope, immutable observations and activation guards.
func TestIngestConstraints(t *testing.T) {
	s := ingestFixture(t)
	require.Equal(t, 90, scalar(t, s, `SELECT details_retention_days FROM integrations WHERE id='source'`))
	require.Zero(t, scalar(t, s, `SELECT diagnostic_enabled FROM integrations WHERE id='source'`))
	for _, q := range []string{
		`UPDATE integrations SET details_retention_days=6`, `UPDATE integrations SET details_retention_days=3651`,
		`UPDATE ingest_events SET source_instance_id='other'`,
		`UPDATE ingest_events SET external_id_trusted=1`,
		`UPDATE ingest_events SET client_idempotency_key='key'`,
		`UPDATE ingest_events SET file_digest='bad'`,
		`UPDATE ingest_events SET event_id='track-1-at-second-1'`,
		`UPDATE ingest_events SET occurrence_id=1`, `UPDATE ingest_events SET recording_id=1`,
		`UPDATE ingest_events SET resolution_version=1`, `UPDATE ingest_events SET state='accepted'`,
		`UPDATE ingest_events SET original_snapshot='{"title":"changed"}'`,
		`INSERT INTO ingest_event_details(profile_id,receipt_id,resolution_version,kind,payload) VALUES(2,'receipt',0,'error','{}')`,
		`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at) VALUES('o',1,'source','future','listen','key',0)`,
		`INSERT INTO ingest_revisions VALUES(1,'receipt',1,'user','review','["conflict"]','{}','1',0)`,
	} {
		_, err := s.db.Exec(q)
		require.Error(t, err, q)
	}
	// Missing optional keys allow indistinguishable receipts, never collapsing by second.
	insertReceipt(t, s, "second")
	execSQL(t, s, `UPDATE ingest_events SET external_id_trusted=1,external_namespace='test:v1',external_event_id='e' WHERE receipt_id='receipt'`)
	_, err := s.db.Exec(`UPDATE ingest_events SET external_id_trusted=1,external_namespace='test:v1',external_event_id='e' WHERE receipt_id='second'`)
	require.Error(t, err)
	execSQL(t, s, `UPDATE ingest_events SET client_idempotency_key='k',batch_digest='b' WHERE receipt_id='receipt'`)
	_, err = s.db.Exec(`UPDATE ingest_events SET client_idempotency_key='k',batch_digest='b' WHERE receipt_id='second'`)
	require.Error(t, err)
	execSQL(t, s, `UPDATE ingest_events SET file_digest=printf('%064d',0),file_format='test',format_version='1' WHERE receipt_id='receipt'`)
	_, err = s.db.Exec(`UPDATE ingest_events SET file_digest=printf('%064d',0),file_format='test',format_version='1' WHERE receipt_id='second'`)
	require.Error(t, err)
	execSQL(t, s, `UPDATE ingest_events SET item_ordinal=1,file_digest=printf('%064d',0),file_format='test',format_version='1' WHERE receipt_id='second'`)
	execSQL(t, s, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at) VALUES('v',1,'receipt',1,'variant','1','{}',0)`)
	_, err = s.db.Exec(`UPDATE ingest_observations SET original_snapshot='{"changed":true}'`)
	require.Error(t, err)
	_, err = s.db.Exec(`DELETE FROM integrations WHERE id='source'`)
	require.Error(t, err)
	execSQL(t, s, `UPDATE integrations SET revoked_at=1,token_ref=NULL WHERE id='source'`)
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM ingest_events`))
	// Removing a profile removes all its receipt keys, including dependent variants.
	execSQL(t, s, `DELETE FROM listening_profiles WHERE id=1`)
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM ingest_events`))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM ingest_observations`))
	require.NoError(t, checkIntegrity(s.db))
}

// TestIngestRetention protects open work, current evidence and snapshots at the exact cutoff.
func TestIngestRetention(t *testing.T) {
	for _, days := range []int{7, 90, 3650} {
		t.Run(time.Duration(days).String(), func(t *testing.T) {
			s := ingestFixture(t)
			ctx := context.Background()
			now := time.Unix(2000000000, 0)
			cutoff := now.Unix() - int64(days)*86400
			execSQL(t, s, `UPDATE integrations SET details_retention_days=? WHERE id='source'`, days)
			execSQL(t, s, `UPDATE ingest_events SET state='invalid',identity_status='invalid',reason_codes='["invalid"]',rule_version='1',closed_at=1`)
			execSQL(t, s, `INSERT INTO ingest_event_details(id,profile_id,receipt_id,resolution_version,kind,payload,closed_at,protected) VALUES
    (1,1,'receipt',0,'error','{}',?,0),(2,1,'receipt',0,'error','{}',?,0),(3,1,'receipt',0,'evidence','{}',?,1),(4,1,'receipt',0,'open','{}',NULL,0)`, cutoff, cutoff+1, cutoff)
			// Reopening and active leases prevent expiry; reading/retrying never resets detail time.
			execSQL(t, s, `UPDATE ingest_events SET state='pending_identity',closed_at=NULL`)
			count, err := s.PurgeClosedIngestDetails(ctx, now, 500)
			require.NoError(t, err)
			require.Zero(t, count)
			execSQL(t, s, `UPDATE ingest_events SET state='invalid',closed_at=1,lease_owner='worker',lease_until=?`, now.Unix()+1)
			count, err = s.PurgeClosedIngestDetails(ctx, now, 500)
			require.NoError(t, err)
			require.Zero(t, count)
			execSQL(t, s, `UPDATE ingest_events SET lease_owner=NULL,lease_until=NULL`)
			count, err = s.PurgeClosedIngestDetails(ctx, now, 1)
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
			require.Equal(t, 3, scalar(t, s, `SELECT COUNT(*) FROM ingest_event_details`))
			require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM ingest_events WHERE original_snapshot='{}' AND evidence='{}'`))
			count, err = s.PurgeClosedIngestDetails(ctx, now, 500)
			require.NoError(t, err)
			require.Zero(t, count)
			_, err = s.PurgeClosedIngestDetails(ctx, now, 0)
			require.Error(t, err)
			_, err = s.PurgeClosedIngestDetails(ctx, now, 501)
			require.Error(t, err)
		})
	}
}

// TestV4MigrationRollback injects an actual SQL failure after the ledger was created.
func TestV4MigrationRollback(t *testing.T) {
	s, p, _ := migrationDB(t, 3)
	before := legacyDigests(t, s.db)
	execSQL(t, s, `CREATE TABLE ingest_observations(conflict TEXT)`)
	_, err := p.Up(context.Background())
	require.Error(t, err)
	require.Equal(t, 3, scalar(t, s, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1`))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM sqlite_master WHERE name IN ('integrations','ingest_events','ingest_event_details')`))
	for name, want := range before {
		require.Equal(t, want, digestTable(t, s.db, name, want.columns))
	}
	execSQL(t, s, `DROP TABLE ingest_observations`)
	_, err = p.Up(context.Background())
	require.NoError(t, err)
	require.NoError(t, checkDatabase(s.db))
	_, err = p.Down(context.Background())
	require.Error(t, err)
	require.NoError(t, checkDatabase(s.db))
}

// TestBootstrapV3WALRestore verifies backup-before-mutation and independent restore.
func TestBootstrapV3WALRestore(t *testing.T) {
	s, _, source := migrationDB(t, 3)
	seedMusic(t, s)
	execSQL(t, s, `INSERT INTO users(id,username,password,role) VALUES(1,'test',x'00','admin')`)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(1,1,42,'test')`)
	target := filepath.Join(t.TempDir(), "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	conn, err := openExisting(target)
	require.NoError(t, err)
	_, err = conn.Exec(`PRAGMA journal_mode=WAL; INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(1,1,43,'wal')`)
	require.NoError(t, err)
	defer conn.Close()
	before := legacyDigests(t, conn)
	old := bootstrapPhase
	t.Cleanup(func() { bootstrapPhase = old })
	bootstrapPhase = func(phase string) error {
		if phase == "inplace-backup-synced" {
			v, e := schemaVersion(conn)
			require.NoError(t, e)
			require.Equal(t, 3, v)
		}
		return nil
	}
	_, err = bootstrap(filepath.Dir(target), true)
	require.NoError(t, err)
	for name, want := range before {
		require.Equal(t, want, digestTable(t, conn, name, want.columns))
	}
	backups, err := filepath.Glob(target + ".pre-ingest-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	bootstrapPhase = old
	restoreDir := t.TempDir()
	restored := filepath.Join(restoreDir, "tunetrail.db")
	require.NoError(t, copyDatabase(backups[0], restored))
	saved, err := openExisting(restored)
	require.NoError(t, err)
	v, err := schemaVersion(saved)
	require.NoError(t, err)
	require.Equal(t, 3, v)
	require.NoError(t, checkIntegrity(saved))
	require.NoError(t, saved.Close())
	_, err = bootstrap(restoreDir, true)
	require.NoError(t, err)
}

// TestBootstrapIncompleteV4 catches removed tables, constraints, queue indexes and guards.
func TestBootstrapIncompleteV4(t *testing.T) {
	for _, damage := range []string{`DROP TABLE outbox_details`, `DROP INDEX idx_ingest_external_key`, `DROP TRIGGER trg_outbox_inactive`, `ALTER TABLE integrations ADD COLUMN unexpected TEXT`} {
		t.Run(damage, func(t *testing.T) {
			dir := t.TempDir()
			path, err := bootstrap(dir, true)
			require.NoError(t, err)
			conn, err := sql.Open("sqlite", path)
			require.NoError(t, err)
			_, err = conn.Exec(damage)
			require.NoError(t, err)
			require.NoError(t, conn.Close())
			_, err = bootstrap(dir, true)
			require.ErrorContains(t, err, "incompatible v4 schema")
		})
	}
	// An incomplete current v3 must be rejected before any backup is created.
	s, _, source := migrationDB(t, 3)
	execSQL(t, s, `DROP TRIGGER trg_api_key_profile_immutable`)
	require.NoError(t, s.db.Close())
	target := filepath.Join(filepath.Dir(source), "tunetrail.db")
	require.NoError(t, os.Rename(source, target))
	_, err := bootstrap(filepath.Dir(target), true)
	require.ErrorContains(t, err, "required trigger")
	files, err := filepath.Glob(target + ".pre-ingest-*")
	require.NoError(t, err)
	require.Empty(t, files)
}

// TestIngestConcurrentTrustedReceipt relies on SQLite constraints across connections,
// not a process-local mutex, to admit exactly one live trusted receipt key.
func TestIngestConcurrentTrustedReceipt(t *testing.T) {
	s, _, path := migrationDB(t, 4)
	execSQL(t, s, `PRAGMA busy_timeout=5000`)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'one','One',0)`)
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0)`)
	other, err := openExisting(path)
	require.NoError(t, err)
	defer other.Close()
	ready := make(chan struct{})
	results := make(chan error, 2)
	for i, conn := range []*sql.DB{s.db, other} {
		go func(id int, conn *sql.DB) {
			<-ready
			_, err := conn.Exec(`INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,external_id_trusted,external_namespace,external_event_id) VALUES(?,1,'source',1,'test','1','batch',0,'submit',0,'second','start','{}','digest','1',1,'test:v1','event')`, fmt.Sprint(id))
			results <- err
		}(i, conn)
	}
	close(ready)
	failures := 0
	for range 2 {
		if err := <-results; err != nil {
			failures++
			require.ErrorContains(t, err, "UNIQUE constraint failed")
		}
	}
	require.Equal(t, 1, failures)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM ingest_events`))
}

// TestOutboxDetailRetention exercises the future queue shape without enabling a
// production writer: only this isolated fixture removes the v4 activation guard.
func TestOutboxDetailRetention(t *testing.T) {
	s := ingestFixture(t)
	now := time.Unix(2000000000, 0)
	cutoff := now.Unix() - 90*86400
	execSQL(t, s, `DROP TRIGGER trg_outbox_inactive`)
	execSQL(t, s, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,state,closed_at) VALUES
 ('sent',1,'source','e1','listen','k1',0,'sent',1),('open',1,'source','e2','listen','k2',0,'failed',NULL)`)
	execSQL(t, s, `INSERT INTO outbox_details(id,profile_id,outbox_id,payload,closed_at,protected) VALUES
 (1,1,'sent','{}',?,0),(2,1,'open','{}',?,0),(3,1,'sent','{}',?,1)`, cutoff, cutoff, cutoff)
	count, err := s.PurgeClosedIngestDetails(context.Background(), now, 500)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM outbox_details`))
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM outbox`))
}

// TestObservationRetention expires only closed variants not needed by the current decision.
func TestObservationRetention(t *testing.T) {
	s := ingestFixture(t)
	now := time.Unix(2000000000, 0)
	cutoff := now.Unix() - 90*86400
	execSQL(t, s, `UPDATE ingest_events SET state='invalid',reason_codes='["invalid"]',rule_version='1',closed_at=1`)
	execSQL(t, s, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at,closed_at,protected) VALUES
 ('expired',1,'receipt',1,'expired','1','{}',0,?,0),('protected',1,'receipt',1,'protected','1','{}',0,?,1),('open',1,'receipt',1,'open','1','{}',0,NULL,0)`, cutoff, cutoff)
	count, err := s.PurgeClosedIngestDetails(context.Background(), now, 500)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM ingest_observations`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM ingest_events`))
}
