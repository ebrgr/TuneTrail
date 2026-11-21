package sqlite

import (
	"bytes"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// admitRetryFixture commits a receipt through the same admission used before an ACK.
func admitRetryFixture(t *testing.T, s *Sqlite, now time.Time, key string) string {
	t.Helper()
	active, results, err := s.AdmitIngest(t.Context(), ingest.Batch{
		ProfileID: 1, Source: "file", Operation: "import", ClientKey: key, ReceivedAt: now,
		Items: []ingest.Item{{ListenedAt: time.Unix(100, 0), Snapshot: ingest.Snapshot{Artist: "A", Title: "T", CatalogTrackID: 1}}},
	}, true, &ingest.JobMode{Active: true, Variant: "v9"}, nil)
	require.NoError(t, err)
	require.True(t, active)
	require.Len(t, results, 1)
	return results[0].ReceiptID
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
			seedIdentityHistory(t, s)
			now := time.Now().Truncate(time.Second)
			s.ingestClock = func() time.Time { return now }
			id := admitRetryFixture(t, s, now, "retry-key")
			var original string
			require.NoError(t, s.db.QueryRow(`SELECT original_snapshot FROM ingest_events WHERE receipt_id=?`, id).Scan(&original))

			execSQL(t, s, `CREATE TRIGGER fixture_resolver_failure BEFORE INSERT ON ingest_revisions
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
					go func() { done <- store.ReprocessIngest(ctx, 1, id) }()
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
			crashed.track, crashed.unix = 1, 100
			require.ErrorIs(t, s.finishIngest(ctx, crashed), sql.ErrNoRows)
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
			s, _, _ := migrationDB(t, currentSchemaVersion)
			seedIdentityHistory(t, s)
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
	id := admitRetryFixture(t, old, time.Now(), "v12-key")
	execSQL(t, old, `UPDATE ingest_events SET state='retryable',attempt_count=3,next_attempt_at=0 WHERE receipt_id=?`, id)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrap(dir, false)
	require.NoError(t, err)
	conn, err := openExisting(target)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, checkDatabase(conn))
	migrated := &Sqlite{db: conn}
	version, err := schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, currentSchemaVersion, version)
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
