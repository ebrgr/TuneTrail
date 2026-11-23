package sqlite

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// admitRetryFixture commits a receipt through the same admission used before an ACK.
func admitRetryFixture(t *testing.T, s *Sqlite, now time.Time, key string) string {
	t.Helper()
	results, err := s.AdmitOccurrence(t.Context(), ingest.Batch{
		ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: key, ReceivedAt: now,
		Items: []ingest.Item{{ListenedAt: time.Unix(100, 0), Snapshot: ingest.Snapshot{Artist: "A", Title: "T", Release: "Synthetic Album"}}},
	})
	require.NoError(t, err)
	require.Len(t, results, 1)
	return results[0].ReceiptID
}

// seedHistoricalRetryReceipt inserts an already-admitted v9 receipt in a migration fixture.
func seedHistoricalRetryReceipt(t *testing.T, s *Sqlite, key string) string {
	t.Helper()
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at)
		VALUES('listenbrainz:1',1,'listenbrainz',0) ON CONFLICT DO NOTHING`)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,
		adapter_version,batch_id,item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,
		original_snapshot,snapshot_digest,digest_version,mode,client_idempotency_key,batch_digest)
		VALUES(?,1,'listenbrainz:1',1,'listenbrainz','1',?,0,'single',0,?,'second','start',
		'{"artist":"A","title":"T"}','fixture','1','active',?,'fixture-batch')`, key, key, time.Unix(100, 0).UTC().Format(receiptTime), key)
	return key
}

func TestIngestRetryBudgetAndRecovery(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "transient"
		if persistent {
			name = "persistent"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			s, _, path := migrationDB(t, currentSchemaVersion)
			testfixtures.Profile(t, s, 1, "retry", "Retry", 0)
			now := time.Now().Truncate(time.Second)
			s.ingestClock = func() time.Time { return now }
			id := admitRetryFixture(t, s, now, "retry-key")
			var original string
			require.NoError(t, s.db.QueryRow(`SELECT original_snapshot FROM ingest_events WHERE receipt_id=?`, id).Scan(&original))

			execSQL(t, s, `CREATE TRIGGER fixture_resolver_failure BEFORE INSERT ON ingest_revisions WHEN NEW.author='automatic'
				BEGIN SELECT RAISE(ABORT, 'fixture revision failure'); END`)
			for i, delay := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute} {
				found, err := s.ResolveNextIngest(ctx, nil)
				require.True(t, found)
				require.ErrorContains(t, err, "fixture revision failure")
				require.Equal(t, i+1, scalar(t, s, `SELECT attempt_count FROM ingest_events WHERE receipt_id=? AND parked_at IS NULL`, id))
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE listened_at=100`))
				require.EqualValues(t, now.Add(delay).Unix(), scalar(t, s, `SELECT next_attempt_at FROM ingest_events WHERE receipt_id=?`, id))
				now = now.Add(delay - time.Second)
				found, err = s.ResolveNextIngest(ctx, nil)
				require.NoError(t, err)
				require.False(t, found)
				now = now.Add(time.Second)
			}

			if persistent {
				found, err := s.ResolveNextIngest(ctx, nil)
				require.True(t, found)
				require.ErrorContains(t, err, "fixture revision failure")
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND
					state='retryable' AND parked_at=? AND park_reason='retry_exhausted' AND client_idempotency_key='retry-key'
					AND original_snapshot=? AND attempt_count=5 AND lease_owner IS NULL AND next_attempt_at IS NULL`, id, now.Unix(), original))
				now = now.Add(time.Hour)
				found, err = s.ResolveNextIngest(ctx, nil)
				require.NoError(t, err)
				require.False(t, found)

				// Separate connections exercise SQLite writer arbitration with the runtime busy timeout.
				execSQL(t, s, `PRAGMA busy_timeout=5000`)
				conn, err := openExisting(path)
				require.NoError(t, err)
				defer conn.Close()
				worker := &Sqlite{db: conn, ingestClock: s.ingestClock}
				done := make(chan error, 2)
				for _, store := range []*Sqlite{s, worker} {
					go func() { done <- store.ReprocessIngest(ctx, 1, 1, id) }()
				}
				for range 2 {
					require.NoError(t, <-done)
				}
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND parked_at IS NULL
					AND park_reason IS NULL AND next_attempt_at=? AND retry_attempt_base=5 AND attempt_count=5 AND work_version=6`, id, now.Unix()))
			}

			execSQL(t, s, `DROP TRIGGER fixture_resolver_failure`)
			crashed, _, _, err := s.claimNextIngest(ctx)
			require.NoError(t, err)
			require.Equal(t, id, crashed.id)
			now = now.Add(119 * time.Second)
			found, err := s.ResolveNextIngest(ctx, nil)
			require.NoError(t, err)
			require.False(t, found)
			now = now.Add(time.Second)
			// A fresh worker resumes the interrupted attempt at the lease boundary.
			worker := &Sqlite{db: s.db, ingestClock: s.ingestClock}
			found, err = worker.ResolveNextIngest(ctx, nil)
			require.NoError(t, err)
			require.True(t, found)
			crashed.unix = 100
			require.NoError(t, s.finishOccurrence(ctx, crashed, occurrenceProof{}))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN listens l USING(profile_id,event_id)
				WHERE e.receipt_id=? AND e.state='accepted' AND l.listened_at=100`, id))
			require.Equal(t, int(crashed.attempts), scalar(t, s, `SELECT attempt_count-retry_attempt_base FROM ingest_events WHERE receipt_id=?`, id))
			require.Equal(t, id, admitRetryFixture(t, s, now, "retry-key"))
		})
	}
}

func TestIngestRetryRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct{ name, column, value, reason string }{
		{"unreadable timestamp", "listened_at", "unreadable", "invalid_listened_at"},
		{"wrong snapshot field type", "original_snapshot", `{"duration_ms":"invalid"}`, "invalid_snapshot"},
		{"malformed snapshot JSON", "original_snapshot", `{"duration_ms":`, "invalid_snapshot"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := migrationDB(t, 21)
			testfixtures.Profile(t, s, 1, "retry", "Retry", 0)
			id := admitRetryFixture(t, s, time.Now(), tc.reason)
			// Simulate a malformed historical receipt beyond the normal writer guards.
			execSQL(t, s, `DROP TRIGGER trg_ingest_snapshot_immutable; PRAGMA ignore_check_constraints=ON`)
			execSQL(t, s, `UPDATE ingest_events SET `+tc.column+`=? WHERE receipt_id=?`, tc.value, id)
			execSQL(t, s, `PRAGMA ignore_check_constraints=OFF`)
			found, err := s.ResolveNextIngest(t.Context(), nil)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e
				JOIN ingest_revisions r ON r.profile_id=e.profile_id AND r.receipt_id=e.receipt_id
				WHERE e.receipt_id=? AND e.state='invalid' AND e.closed_at IS NOT NULL AND e.attempt_count=1
				AND e.rule_version=? AND e.reason_codes=json_array(?) AND r.decision='invalid'`, id, activeRuleVersion, tc.reason))
		})
	}
}

func TestIngestRetryUpgradeFromV12(t *testing.T) {
	old, _, source := migrationDB(t, 12)
	seedIdentityHistory(t, old)
	id := seedHistoricalRetryReceipt(t, old, "v12-key")
	execSQL(t, old, `UPDATE ingest_events SET state='retryable',attempt_count=3,next_attempt_at=0 WHERE receipt_id=?`, id)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrapAtVersion(dir, false, 20)
	require.NoError(t, err)
	conn, err := openExisting(target)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, checkDatabase(conn))
	migrated := &Sqlite{db: conn}
	version, err := schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, 20, version)
	require.Equal(t, 1, scalar(t, migrated, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND
		client_idempotency_key='v12-key' AND attempt_count=3 AND retry_attempt_base=0 AND parked_at IS NULL`, id))
	claimed, _, _, err := migrated.claimNextIngest(t.Context())
	require.NoError(t, err)
	require.Equal(t, id, claimed.id)
	require.EqualValues(t, 4, claimed.attempts)
	backups, err := filepath.Glob(target + ".pre-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	defer backup.Close()
	version, err = schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 12, version)
}

func TestIngestParkWarningRate(t *testing.T) {
	var output bytes.Buffer
	log := zerolog.New(&output)
	ctx := logger.NewContext(&log)
	s := &Sqlite{}
	now := time.Now()
	key := ingestAlertKey{profile: 1, reason: ingestRetryExhausted}
	s.warnParkedIngest(ctx, key, now)
	s.warnParkedIngest(ctx, key, now.Add(59*time.Second))
	s.warnParkedIngest(ctx, ingestAlertKey{profile: 2, reason: ingestRetryExhausted}, now.Add(59*time.Second))
	s.warnParkedIngest(ctx, key, now.Add(time.Minute))
	require.Equal(t, 3, bytes.Count(output.Bytes(), []byte(`"level":"warn"`)))
	require.Equal(t, 2, bytes.Count(output.Bytes(), []byte(`"profile_id":1,"reason":"retry_exhausted"`)))
	require.Equal(t, 1, bytes.Count(output.Bytes(), []byte(`"profile_id":2,"reason":"retry_exhausted"`)))
}

// TestIngestLeaseClaimCap makes every resolution cross the lease boundary using the scheduler clock.
func TestIngestLeaseClaimCap(t *testing.T) {
	s, _, _ := migrationDB(t, 21)
	testfixtures.Profile(t, s, 1, "retry", "Retry", 0)
	now := time.Now().Truncate(time.Second)
	id := admitRetryFixture(t, s, now, "lease-cap")
	var snapshot string
	require.NoError(t, s.db.QueryRow(`SELECT original_snapshot FROM ingest_events WHERE receipt_id=?`, id).Scan(&snapshot))
	var output bytes.Buffer
	log := zerolog.New(&output)
	ctx := logger.NewContext(&log)

	total := 0
	for cycle, budget := range []int{maxIngestClaims, maxIngestAttempts} {
		total += budget
		for range budget {
			claimTime := now
			s.ingestClock = func() time.Time {
				current := now
				now = claimTime.Add(120 * time.Second)
				return current
			}
			found, err := s.ResolveNextIngest(ctx, nil)
			require.True(t, found)
			require.NoError(t, err)
		}
		s.ingestClock = func() time.Time { return now }
		found, err := s.ResolveNextIngest(ctx, nil)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
			AND parked_at=? AND park_reason='lease_exhausted' AND lease_owner IS NULL AND lease_until IS NULL
			AND next_attempt_at IS NULL AND attempt_count=? AND attempt_count-retry_attempt_base=1
			AND original_snapshot=? AND client_idempotency_key='lease-cap'`, id, now.Unix(), total, snapshot))
		found, err = s.ResolveNextIngest(ctx, nil)
		require.NoError(t, err)
		require.False(t, found)
		if cycle == 0 {
			require.NoError(t, s.ReprocessIngest(ctx, 1, 1, id))
		}
	}
	require.Equal(t, 2, bytes.Count(output.Bytes(), []byte(`"reason":"lease_exhausted"`)))
	require.Equal(t, id, admitRetryFixture(t, s, now, "lease-cap"))
}

// TestIngestReprocessAllWithResolver lets two connections contend for the active writer.
func TestIngestReprocessAllWithResolver(t *testing.T) {
	ctx := t.Context()
	s, _, path := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, s, 1, "retry", "Retry", 0)
	now := time.Now().Truncate(time.Second)
	s.ingestClock = func() time.Time { return now }
	id := admitRetryFixture(t, s, now, "bulk-race")
	execSQL(t, s, `UPDATE ingest_events SET state='retryable',attempt_count=5,work_version=5,
		parked_at=1,park_reason='retry_exhausted' WHERE receipt_id=?`, id)
	execSQL(t, s, `PRAGMA busy_timeout=5000`)
	conn, err := openExisting(path)
	require.NoError(t, err)
	defer conn.Close()
	worker := &Sqlite{db: conn, ingestClock: s.ingestClock}

	start := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		<-start
		done <- s.ReprocessAllIngest(ctx, 1, 7)
	}()
	go func() {
		<-start
		_, err := worker.ResolveNextIngest(ctx, nil)
		done <- err
	}()
	close(start)
	// Drain both goroutines before testing their results, even if one failed.
	first, second := <-done, <-done
	require.NoError(t, first)
	require.NoError(t, second)
	_, err = worker.ResolveNextIngest(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, s.ReprocessAllIngest(ctx, 1, 8))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND state='accepted' AND attempt_count=6 AND work_version=7 AND claim_attempt_base=5
		AND parked_at IS NULL AND reprocessed_at=? AND reprocessed_by=7`, id, now.Unix()))
}

// TestIngestReprocessUpgradeFromV17 checks the pre-upgrade backup and the downgrade guard.
func TestIngestReprocessUpgradeFromV17(t *testing.T) {
	old, _, source := migrationDB(t, 17)
	seedIdentityHistory(t, old)
	id := seedHistoricalRetryReceipt(t, old, "v17-key")
	execSQL(t, old, `UPDATE ingest_events SET state='retryable',attempt_count=40,retry_attempt_base=38,
		parked_at=1,park_reason='retry_exhausted' WHERE receipt_id=?`, id)

	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrapAtVersion(dir, false, 20)
	require.NoError(t, err)
	conn, err := openExisting(target)
	require.NoError(t, err)
	defer conn.Close()
	migrated := &Sqlite{db: conn}
	require.Equal(t, 1, scalar(t, migrated, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND
		attempt_count=40 AND claim_attempt_base=40 AND retry_attempt_base=38 AND parked_at=1
		AND client_idempotency_key='v17-key' AND reprocessed_at IS NULL AND reprocessed_by IS NULL`, id))

	backups, err := filepath.Glob(target + ".pre-ingest-reprocess-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	defer backup.Close()
	version, err := schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 17, version)

	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrations.Files)
	require.NoError(t, err)
	_, err = provider.Down(t.Context())
	require.ErrorContains(t, err, "CHECK constraint failed")
	version, err = schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, 20, version)
	require.NoError(t, checkDatabase(conn))
}

// TestIngestReprocessAllLargeQueue verifies multiple batches and reports the writer transaction's wall time.
func TestIngestReprocessAllLargeQueue(t *testing.T) {
	s, _, _ := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	execSQL(t, s, `WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<10000)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,state,attempt_count,work_version,parked_at,park_reason)
		SELECT 'parked-'||n,1,'source',1,'test','1','batch',n,'single',n,'1970-01-01T00:00:01.000000000Z','second','start',
			'{"artist":"Fixture","title":"Track","client":"test"}','digest','1',
			'active','retryable',5,5,n,'retry_exhausted' FROM sample`)
	start := time.Now()
	require.NoError(t, s.ReprocessAllIngest(t.Context(), 1, 1))
	t.Logf("Reprocessing 10000 parked receipts took %s", time.Since(start))
	require.NoError(t, s.ReprocessAllIngest(t.Context(), 1, 2))
	require.Equal(t, 10000, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1
		AND parked_at IS NULL AND park_reason IS NULL AND attempt_count=5 AND work_version=6
		AND claim_attempt_base=5 AND retry_attempt_base=5 AND reprocessed_by=1 AND reprocessed_at=next_attempt_at`))
	require.Equal(t, 10000, scalar(t, s, `SELECT count(*) FROM ingest_revisions
		WHERE profile_id=1 AND author='user' AND reason_codes=json_array('reprocessed')`))
}

type blockingReleaseCaller struct {
	mbz.MbzErrorCaller
	onCall func()
}

// GetRelease blocks like an upstream request that responds only to cancellation.
func (m blockingReleaseCaller) GetRelease(ctx context.Context, _ uuid.UUID) (*mbz.MusicBrainzRelease, error) {
	m.onCall()
	<-ctx.Done()
	return nil, ctx.Err()
}

// TestIngestAttemptDeadline verifies retry writes after timeout and progress on the next receipt.
func TestIngestAttemptDeadline(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	now := time.Now().Truncate(time.Second)
	s.ingestClock = func() time.Time { return now }
	snapshot := ingest.Snapshot{Artist: "Artist", Title: "Slow", Release: "Album", ReleaseMBID: uuid.NewString()}
	slow := admitOccurrenceFixture(t, s, snapshot, 100, "slow")
	calls := 0
	client := &blockingReleaseCaller{onCall: func() { calls++ }}
	const timeout = 100 * time.Millisecond

	for attempt, delay := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 0} {
		found, err := s.resolveNextIngest(t.Context(), client, timeout)
		require.True(t, found)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, attempt+1, calls)
		if delay == 0 {
			break
		}
		require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
			AND state='retryable' AND attempt_count=? AND next_attempt_at=? AND lease_owner IS NULL`,
			slow, attempt+1, now.Add(delay).Unix()))
		if attempt == 0 {
			next := admitOccurrenceFixture(t, s, ingest.Snapshot{Artist: "Artist", Title: "Next", Release: "Album"}, 101, "next")
			found, err = s.ResolveNextIngest(t.Context(), client)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND state='accepted'`, next))
		}
		now = now.Add(delay)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND parked_at=? AND park_reason='retry_exhausted' AND attempt_count=5 AND lease_owner IS NULL`, slow, now.Unix()))
}

// TestIngestAttemptShutdown leaves the interrupted attempt recoverable without spending a retry.
func TestIngestAttemptShutdown(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	now := time.Now().Truncate(time.Second)
	s.ingestClock = func() time.Time { return now }
	snapshot := ingest.Snapshot{Artist: "Artist", Title: "Interrupted", Release: "Album", ReleaseMBID: uuid.NewString()}
	id := admitOccurrenceFixture(t, s, snapshot, 100, "shutdown")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	found, err := s.ResolveNextIngest(ctx, &blockingReleaseCaller{onCall: cancel})
	require.True(t, found)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND attempt_count=1 AND state='received' AND lease_owner IS NOT NULL AND parked_at IS NULL AND next_attempt_at IS NULL`, id))
	now = now.Add(ingestLease)
	found, err = s.ResolveNextIngest(t.Context(), nil)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND state='accepted' AND attempt_count=2 AND retry_attempt_base=1`, id))
}

// TestIngestReprocessHistory exercises both actions, both reasons and the profile boundary.
func TestIngestReprocessHistory(t *testing.T) {
	s, _, _ := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	ctx := t.Context()
	now := time.Now().Truncate(time.Second)
	s.ingestClock = func() time.Time { return now }
	first := seedHistoricalRetryReceipt(t, s, "short-budget")
	second := seedHistoricalRetryReceipt(t, s, "full-budget")
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('other',2,'test',0);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,state,attempt_count,parked_at,park_reason)
		VALUES('other-profile',2,'other',1,'test','1','batch',0,'single',0,'1970-01-01T00:00:01.000000000Z','second','start',
			'{}','digest','1','active','retryable',25,1,'lease_exhausted')`)
	execSQL(t, s, `UPDATE ingest_events SET state='retryable',attempt_count=25,parked_at=1,park_reason='lease_exhausted' WHERE receipt_id=?`, first)
	execSQL(t, s, `UPDATE ingest_events SET state='retryable',attempt_count=5,parked_at=1,park_reason='retry_exhausted' WHERE receipt_id=?`, second)
	require.NoError(t, s.ReprocessIngest(ctx, 1, 7, "other-profile"))
	require.NoError(t, s.ReprocessIngest(ctx, 1, 7, first))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND attempt_count-claim_attempt_base=20 AND retry_attempt_base=attempt_count AND resolution_version=1`, first))
	firstAt := now.Unix()

	// Simulate another round of expired claims before the bulk action.
	now = now.Add(time.Hour)
	execSQL(t, s, `UPDATE ingest_events SET attempt_count=30,parked_at=2,park_reason='lease_exhausted' WHERE receipt_id=?`, first)
	require.NoError(t, s.ReprocessAllIngest(ctx, 1, 8))
	require.NoError(t, s.ReprocessAllIngest(ctx, 1, 9))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND attempt_count-claim_attempt_base=20 AND retry_attempt_base=attempt_count
		AND resolution_version=2 AND reprocessed_by=8 AND reprocessed_at=?`, first, now.Unix()))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND claim_attempt_base=attempt_count AND retry_attempt_base=attempt_count AND resolution_version=1`, second))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='other-profile'
		AND parked_at=1 AND claim_attempt_base=0 AND resolution_version=0 AND reprocessed_at IS NULL`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE profile_id=2`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id=? AND author='user'
		AND reason_codes=json_array('reprocessed') AND (
			(resolution_version=1 AND json_extract(evidence,'$.decided_by')=7 AND created_at=?) OR
			(resolution_version=2 AND json_extract(evidence,'$.decided_by')=8 AND created_at=?))`, first, firstAt, now.Unix()))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id=?`, first))
}
