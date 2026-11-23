package sqlite

import (
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestIngestDiscardStates covers every eligible state, the audit, and both delete projections.
func TestIngestDiscardStates(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	now := time.Unix(1714564800, 0)
	s.ingestClock = func() time.Time { return now }
	relayReceipt(t, s, 2, "other", now)
	for i, state := range []string{"received", "pending_identity", "pending_duplicate", "retryable", "invalid", "parked"} {
		t.Run(state, func(t *testing.T) {
			relayReceipt(t, s, 1, state, now.Add(time.Duration(i+1)*time.Second))
			if state == "parked" {
				execSQL(t, s, `UPDATE ingest_events SET state='retryable',parked_at=?,park_reason='retry_exhausted',
					reason_codes='["fixture_reason"]' WHERE receipt_id=?`, now.Unix(), state)
			} else if state == "invalid" {
				execSQL(t, s, `UPDATE ingest_events SET state='invalid',closed_at=?,rule_version='fixture',
					reason_codes='["fixture_reason"]' WHERE receipt_id=?`, now.Unix(), state)
			} else {
				execSQL(t, s, `UPDATE ingest_events SET state=?,rule_version='fixture',reason_codes='["fixture_reason"]' WHERE receipt_id=?`, state, state)
			}
			execSQL(t, s, `INSERT INTO ingest_event_details(profile_id,receipt_id,resolution_version,kind,payload)
				VALUES(1,?,0,'test','{"body":"private"}')`, state)

			preview, err := s.IngestDiscardPreview(ctx, 1, state, 1)
			require.NoError(t, err)
			require.Len(t, preview.Receipts, 1)
			digest := preview.Receipts[0].Digest
			require.ErrorIs(t, s.DiscardIngestPending(ctx, 2, 7, state, digest), db.ErrNotFound)
			require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, state, digest))
			require.NoError(t, s.DiscardIngestPending(ctx, 1, 8, state, digest))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_discards WHERE profile_id=1
				AND receipt_id=? AND actor_id=7 AND discarded_at=? AND reason_codes='["fixture_reason"]'
				AND origin_state=CASE WHEN ?='parked' THEN 'retryable' ELSE ? END
				AND park_reason IS CASE WHEN ?='parked' THEN 'retry_exhausted' END`, state, now.Unix(), state, state, state))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, state))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_event_details WHERE receipt_id=?`, state))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=1`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_inbox_daily WHERE profile_id=1`))
		})
	}
	require.Equal(t, 8, scalar(t, s, `SELECT count(*) FROM pragma_table_info('ingest_discards')
		WHERE name IN ('id','profile_id','receipt_id','actor_id','discarded_at','origin_state','park_reason','reason_codes')`))
	require.Equal(t, 8, scalar(t, s, `SELECT count(*) FROM pragma_table_info('ingest_discards')`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT sum(total) FROM ingest_inbox_daily WHERE profile_id=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=2`))
}

// TestIngestDiscardProtectsListens covers duplicate links and accepted receipts with open relay work.
func TestIngestDiscardProtectsListens(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	instant := time.Unix(101, 0)
	relayReceipt(t, s, 1, "accepted", instant)
	preview, err := s.IngestDiscardPreview(ctx, 1, "accepted", 1)
	require.NoError(t, err)
	stale := preview.Receipts[0].Digest
	resolveOccurrenceFixture(t, s, nil)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM outbox JOIN ingest_events USING(profile_id,event_id)
		WHERE receipt_id='accepted' AND outbox.closed_at IS NULL`))
	relayReceipt(t, s, 1, "linked", instant)
	resolveOccurrenceFixture(t, s, nil)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='linked' AND state='duplicate' AND event_id IS NOT NULL`))
	for _, receipt := range []string{"accepted", "linked"} {
		_, err := s.IngestDiscardPreview(ctx, 1, receipt, 1)
		require.ErrorIs(t, err, db.ErrIngestDiscardListen)
		require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, receipt, stale), db.ErrIngestDiscardListen)
	}
	preview, err = s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.Empty(t, preview.Receipts)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM outbox WHERE closed_at IS NULL`))
}

// TestIngestDiscardWriterRace exercises the actual competing mutations on independent connections.
func TestIngestDiscardWriterRace(t *testing.T) {
	for _, kind := range []string{"claim", "review", "reprocess"} {
		t.Run(kind, func(t *testing.T) {
			s, path := relayStoreFixture(t)
			ctx := t.Context()
			execSQL(t, s, `PRAGMA busy_timeout=5000`)
			require.NoError(t, s.SetIngestReview(t.Context(), 1, true))
			relayReceipt(t, s, 1, "race", time.Unix(42, 0))
			switch kind {
			case "review":
				resolveOccurrenceFixture(t, s, nil)
			case "reprocess":
				execSQL(t, s, `UPDATE ingest_events SET state='retryable',parked_at=1,park_reason='retry_exhausted' WHERE receipt_id='race'`)
			}
			preview, err := s.IngestDiscardPreview(ctx, 1, "race", 1)
			require.NoError(t, err)
			require.Len(t, preview.Receipts, 1)
			health, err := s.IngestHealth(ctx, 1)
			require.NoError(t, err)
			conn, err := openExisting(path)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })
			other := &Sqlite{db: conn}

			start := make(chan struct{})
			discarded, competed := make(chan error, 1), make(chan error, 1)
			go func() {
				<-start
				discarded <- s.DiscardIngestPending(ctx, 1, 7, "race", preview.Receipts[0].Digest)
			}()
			go func() {
				<-start
				switch kind {
				case "review":
					competed <- other.ResolveIngestReviewBatch(ctx, 1, 8, "same", health.ReviewSame.Digest)
				case "reprocess":
					competed <- other.ReprocessIngest(ctx, 1, 8, "race")
				default:
					_, _, _, err := other.claimNextIngest(ctx)
					competed <- err
				}
			}()
			close(start)
			discardErr, competeErr := <-discarded, <-competed
			if discardErr == nil {
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_discards WHERE receipt_id='race'`))
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='race'`))
				if kind == "review" {
					require.ErrorIs(t, competeErr, db.ErrConflict)
				} else {
					require.NoError(t, competeErr)
				}
				return
			}
			require.NoError(t, competeErr)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
			switch kind {
			case "review":
				require.ErrorIs(t, discardErr, db.ErrIngestDiscardListen)
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='race' AND event_id IS NOT NULL`))
			case "reprocess":
				require.ErrorIs(t, discardErr, db.ErrConflict)
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='race' AND reprocessed_by=8`))
			default:
				require.ErrorIs(t, discardErr, db.ErrConflict)
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='race' AND lease_until IS NOT NULL`))
			}
		})
	}
}

// TestIngestDiscardBatch checks full preview, batch rollback, and retry safety.
func TestIngestDiscardBatch(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	seedDiscardQueue(t, s, ingestDiscardBatchSize+2, "occurrence")
	latest, err := s.IngestDiscardPreview(ctx, 1, "bulk-00502", 1)
	require.NoError(t, err)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, "bulk-00502", latest.Receipts[0].Digest))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=1 AND receipt_id='bulk-00501'`))
	p, err := s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.Equal(t, ingestDiscardBatchSize+1, p.Batch.Count)
	require.Len(t, p.Receipts, ingestDiscardPageSize)
	require.Equal(t, 2, p.Next)
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 2, 7, "", p.Batch.Digest), db.ErrConflict)
	execSQL(t, s, `CREATE TRIGGER fail_discard BEFORE DELETE ON ingest_events WHEN OLD.receipt_id='bulk-00501'
		BEGIN SELECT RAISE(ABORT,'discard rollback fixture'); END`)
	require.ErrorContains(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest), "discard rollback fixture")
	// The first transaction survived. The failed transaction left neither a deletion nor audit.
	require.Equal(t, ingestDiscardBatchSize+1, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
	require.Equal(t, 1, scalar(t, s, `SELECT sum(total) FROM ingest_inbox_daily WHERE profile_id=1`))
	execSQL(t, s, `DROP TRIGGER fail_discard`)
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest), db.ErrConflict)
	p, err = s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.Equal(t, 1, p.Batch.Count)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest))
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest), db.ErrConflict)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=1`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_inbox_daily WHERE profile_id=1`))
}

// TestIngestDiscardSetDrift makes an unconfirmed receipt eligible at the first batch boundary.
func TestIngestDiscardSetDrift(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	seedDiscardQueue(t, s, ingestDiscardBatchSize+2, "occurrence")
	now := time.Now()
	s.ingestClock = func() time.Time { return now }
	execSQL(t, s, `UPDATE ingest_events SET lease_owner='fixture',lease_until=? WHERE receipt_id='bulk-00502'`, now.Add(time.Second).Unix())
	p, err := s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.Equal(t, ingestDiscardBatchSize+1, p.Batch.Count)
	// The first transaction sees the original clock. The lease expires before the next scan.
	calls := 0
	s.ingestClock = func() time.Time {
		calls++
		return now.Add(time.Duration(calls-1) * time.Second)
	}
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest), db.ErrConflict)
	require.Equal(t, ingestDiscardBatchSize, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id IN ('bulk-00501','bulk-00502')`))
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest), db.ErrConflict)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id IN ('bulk-00501','bulk-00502')`))
}

// TestIngestDiscardTrustedReplay checks repeated discards even within the same second.
func TestIngestDiscardTrustedReplay(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	ctx := t.Context()
	require.NoError(t, s.SetIngestReview(ctx, 1, true))
	now := time.Unix(1714564800, 0)
	s.ingestClock = func() time.Time { return now }
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ReceivedAt: now,
		Items: []ingest.Item{{ExternalNamespace: "fixture", ExternalID: "stable", ListenedAt: time.Unix(5000, 0),
			Snapshot: ingest.Snapshot{Title: "Trusted replay", Artist: "Artist", Release: "Album"}}}}
	admit := func() ingest.Result {
		t.Helper()
		result, err := s.AdmitOccurrence(ctx, batch)
		require.NoError(t, err)
		require.Len(t, result, 1)
		return result[0]
	}
	original := admit()
	resolveOccurrenceFixture(t, s, nil)
	first := admit()
	require.True(t, first.Pending)
	require.NotEqual(t, original.ReceiptID, first.ReceiptID)
	receipt := first.ReceiptID
	p, err := s.IngestDiscardPreview(ctx, 1, receipt, 1)
	require.NoError(t, err)
	old := p.Receipts[0].Digest
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, receipt, old))
	require.Equal(t, receipt, admit().ReceiptID)
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 1, 7, receipt, old), db.ErrConflict)
	p, err = s.IngestDiscardPreview(ctx, 1, receipt, 1)
	require.NoError(t, err)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 8, receipt, p.Receipts[0].Digest))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_discards
		WHERE receipt_id=? AND discarded_at=? AND actor_id IN (7,8)`, receipt, now.Unix()))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=5000`))
}

// TestIngestDiscardRetention preserves the boundary and scopes cleanup to the acting profile.
func TestIngestDiscardRetention(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	now := time.Unix(1734564800, 0)
	s.ingestClock = func() time.Time { return now }
	boundary := now.Add(-ingestDiscardRetention).Unix()
	execSQL(t, s, `INSERT INTO ingest_discards(profile_id,receipt_id,actor_id,discarded_at,origin_state,reason_codes)
		VALUES(1,'expired',7,?,'received','[]'),(1,'boundary',7,?,'received','[]'),(2,'other',7,?,'received','[]')`,
		boundary-1, boundary, boundary-1)
	seedDiscardQueue(t, s, 1, "occurrence")
	p, err := s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_discards WHERE receipt_id='expired'`))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM ingest_discards WHERE receipt_id IN ('boundary','other','bulk-00001')`))
}

// seedDiscardQueue creates one source with monotonically ordered synthetic receipts.
func seedDiscardQueue(t *testing.T, s *Sqlite, count int, variant string) {
	t.Helper()
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('bulk',1,'listenbrainz',0)`)
	execSQL(t, s, `WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<?)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,delivery_policy,resolver_variant)
		SELECT printf('bulk-%05d',i),1,'bulk',1,'listenbrainz','1','bulk',i,'single',1714564800+i,'second','start',
			'{"title":"Discard fixture"}','fixture','1','active',(SELECT policy FROM relay_delivery_gate WHERE id=1),? FROM n`, count, variant)
}

// TestIngestDiscardScale measures the confirmed writer calls for 10,000 receipts.
func TestIngestDiscardScale(t *testing.T) {
	if testing.Short() {
		t.Skip("scale fixture")
	}
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	const size = 10_000
	seedDiscardQueue(t, s, size, "occurrence")
	execSQL(t, s, `UPDATE ingest_events SET state='retryable',attempt_count=5,work_version=5,parked_at=received_at,park_reason='retry_exhausted'
		WHERE profile_id=1`)
	previewStart := time.Now()
	p, err := s.IngestDiscardPreview(ctx, 1, "", 1)
	previewTime := time.Since(previewStart)
	require.NoError(t, err)
	require.Equal(t, size, p.Batch.Count)
	start := time.Now()
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, "", p.Batch.Digest))
	t.Logf("10,000 parked receipts in %d transactions: %s preview, %s writer total",
		size/ingestDiscardBatchSize, previewTime, time.Since(start))
	require.Equal(t, size, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=1`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_inbox_daily WHERE profile_id=1`))
}

// TestIngestDiscardMigration upgrades a populated v19 database and protects the audit on downgrade.
func TestIngestDiscardMigration(t *testing.T) {
	s, provider, _ := migrationDB(t, 19)
	ctx := t.Context()
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id='shadow'`)
	seedDiscardQueue(t, s, 1, "v9")
	_, err := provider.UpTo(ctx, 20)
	require.NoError(t, err)
	preview, err := s.IngestDiscardPreview(ctx, 1, "", 1)
	require.NoError(t, err)
	require.Equal(t, 1, preview.Batch.Count)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, 7, "", preview.Batch.Digest))
	_, err = provider.Down(ctx)
	require.ErrorContains(t, err, "CHECK constraint failed")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
	require.NoError(t, checkDatabase(s.db))
	require.NoError(t, s.PurgeAllData(ctx))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_discards`))
}
