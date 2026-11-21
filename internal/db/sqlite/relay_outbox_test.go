package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

type lookupFixture map[string][]netip.Addr

func (fixture lookupFixture) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	if ips, ok := fixture[host]; ok {
		return ips, nil
	}
	return nil, errors.New("synthetic DNS failure")
}

type slowFirstLookup struct{ fast atomic.Int32 }

func (r *slowFirstLookup) LookupNetIP(ctx context.Context, _, host string) ([]netip.Addr, error) {
	if host == "slow.test" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	r.fast.Add(1)
	return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
}

// TestRelaySeedSlowFirstHost keeps later hosts independent and deduplicates ports.
func TestRelaySeedSlowFirstHost(t *testing.T) {
	seeds := []relaySeed{{raw: "https://slow.test", admin: true}, {raw: "https://fast.test", admin: true}, {raw: "https://FAST.TEST:8443", admin: true}}
	resolver := &slowFirstLookup{}
	resolveRelaySeeds(t.Context(), seeds, resolver)
	require.True(t, seeds[0].blocked)
	require.False(t, seeds[1].blocked)
	require.False(t, seeds[2].blocked)
	require.EqualValues(t, 1, resolver.fast.Load())
}

// TestRelayUpgrade preserves the v11 data and seeds only narrowly scoped exceptions.
func TestRelayUpgrade(t *testing.T) {
	s, _, path := migrationDB(t, 11)
	for i, u := range []string{"HTTPS://PUBLIC.TEST:443/base", "http://clear.test:80", "https://private.test", "https://mixed.test", "https://unknown.test", "http://clear.test", "https://private.test"} {
		id := int32(i + 1)
		testfixtures.Profile(t, s, int(id), fmt.Sprintf("profile%d", id), fmt.Sprintf("Profile %d", id), 0)
		if id <= 5 {
			execSQL(t, s, "UPDATE users SET role='admin' WHERE id=?", id)
		}
		execSQL(t, s, "INSERT INTO profile_relays VALUES(?,?,?)", id, u, "relay-secret-fixture")
	}
	execSQL(t, s, "INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0)")
	insertReceipt(t, s, "upgrade-shadow")
	dns := lookupFixture{
		"public.test":  {netip.MustParseAddr("8.8.8.8")},
		"clear.test":   {netip.MustParseAddr("8.8.4.4")},
		"private.test": {netip.MustParseAddr("10.1.2.3")},
		"mixed.test":   {netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("169.254.170.2")},
	}
	originalHook := bootstrapPhase
	t.Cleanup(func() { bootstrapPhase = originalHook })
	for _, phase := range []string{"relay-before-ddl", "relay-after-ddl", "relay-after-seed", "relay-before-commit"} {
		bootstrapPhase = func(where string) error {
			if where == phase {
				return errors.New("fixture rollback")
			}
			return nil
		}
		require.ErrorContains(t, upgradeRelaySchema(t.Context(), s.db, dns), "fixture rollback")
		require.Equal(t, 11, scalar(t, s, "SELECT max(version_id) FROM goose_db_version"))
		require.Zero(t, scalar(t, s, "SELECT count(*) FROM sqlite_master WHERE name='relay_destinations'"))
		require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM profile_relays WHERE url='HTTPS://PUBLIC.TEST:443/base'"))
	}
	bootstrapPhase = originalHook
	require.NoError(t, upgradeRelaySchema(t.Context(), s.db, dns))
	reopened, err := openExisting(path)
	require.NoError(t, err)
	defer reopened.Close()
	require.NoError(t, upgradeRelaySchema(t.Context(), reopened, dns))
	require.NoError(t, checkDatabase(reopened))
	require.NoError(t, checkDatabase(s.db))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE receipt_id='upgrade-shadow' AND delivery_policy='legacy'"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM profile_relays WHERE url='https://public.test/base'"))
	require.Equal(t, 2, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE origin='seeded'"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE profile_id=2 AND host_port='clear.test:80' AND allow_http=1 AND allow_private=0"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE profile_id=3 AND cidr='10.1.2.3/32' AND allow_private=1 AND allow_http=0"))
	require.Equal(t, 4, scalar(t, s, "SELECT count(*) FROM relay_destinations WHERE profile_id IN (4,5,6,7) AND pause_class='policy_blocked'"))
	for _, table := range []string{"relay_destinations", "relay_allowlist", "outbox", "ingest_events", "integrations"} {
		rows, err := s.db.Query("SELECT * FROM " + table)
		require.NoError(t, err)
		cols, err := rows.Columns()
		require.NoError(t, err)
		for rows.Next() {
			values, pointers := make([]any, len(cols)), make([]any, len(cols))
			for i := range values {
				pointers[i] = &values[i]
			}
			require.NoError(t, rows.Scan(pointers...))
			require.NotContains(t, fmt.Sprint(values), "relay-secret-fixture")
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	_, err = s.db.Exec("UPDATE ingest_events SET delivery_policy='durable'")
	require.Error(t, err)
	execSQL(t, s, "DROP TRIGGER trg_occurrence_relay_update; CREATE TRIGGER trg_occurrence_relay_update AFTER UPDATE ON profile_relays BEGIN SELECT 1; END")
	require.ErrorContains(t, checkDatabase(s.db), "altered trigger trg_occurrence_relay_update")
}

// TestRelayBootstrapWAL proves the v11 backup captures committed WAL data before upgrading.
func TestRelayBootstrapWAL(t *testing.T) {
	s, _, source := migrationDB(t, 11)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	execSQL(t, s, "INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0)")
	insertReceipt(t, s, "wal-only")
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(source + suffix)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(target+suffix, data, 0600))
	}
	file, err := bootstrap(dir, false)
	require.NoError(t, err)
	require.Equal(t, target, file)
	matches, err := filepath.Glob(target + ".pre-relay-delivery-*.bak")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	backup, err := openExisting(matches[0])
	require.NoError(t, err)
	defer backup.Close()
	require.Equal(t, 11, scalar(t, &Sqlite{db: backup}, "SELECT max(version_id) FROM goose_db_version"))
	require.Equal(t, 1, scalar(t, &Sqlite{db: backup}, "SELECT count(*) FROM ingest_events WHERE receipt_id='wal-only'"))
	_, err = bootstrap(dir, false)
	require.NoError(t, err)
	second, err := filepath.Glob(target + ".pre-relay-delivery-*.bak")
	require.NoError(t, err)
	require.Equal(t, matches, second)
}

// TestRelayD4KeepsTheLegacyCut exercises both D4 directions before policy activation.
func TestRelayD4KeepsTheLegacyCut(t *testing.T) {
	s, _, _ := migrationDB(t, 12)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	testfixtures.Profile(t, s, 2, "two", "Two", 0)
	require.NoError(t, s.SetProfileRelay(t.Context(), 1, "https://relay.test", "fixture"))
	_, err := s.db.Exec("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
	require.ErrorContains(t, err, "no relay")
	execSQL(t, s, "INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,0),(2,1,1)")
	_, err = s.db.Exec("UPDATE profile_ingest_flags SET occurrence_write=1 WHERE profile_id=1")
	require.ErrorContains(t, err, "no relay")
	require.Error(t, s.SetProfileRelay(t.Context(), 2, "https://relay.test", "fixture"))
	execSQL(t, s, "UPDATE relay_delivery_gate SET policy='durable'")
	execSQL(t, s, "UPDATE profile_ingest_flags SET occurrence_write=1 WHERE profile_id=1")
	require.NoError(t, s.SetProfileRelay(t.Context(), 2, "https://relay.test", "fixture"))
	_, err = s.db.Exec("UPDATE relay_delivery_gate SET policy='legacy'")
	require.ErrorContains(t, err, "cannot be reversed")
}

// relayReceipt creates an active fixture under the simulated E3 policy.
func relayReceipt(t *testing.T, s *Sqlite, profile int32, id, variant string, track int, instant time.Time) {
	t.Helper()
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('origin:'||?,?,'listenbrainz',0) ON CONFLICT DO NOTHING`, profile, profile)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,delivery_policy)
		VALUES(?,?,'origin:'||?,1,'listenbrainz','1',?,0,'single',? ,?,'second','start',
		json_object('artist','Artist & Co','title','Relay Track','release','Fixture','catalog_track_id',?),'fixture','1','active',?,'durable')`,
		id, profile, profile, id, instant.Unix(), instant.UTC().Format(receiptTime), track, variant)
}

// relayStoreFixture supplies catalog rows and profiles for the two independent writers.
func relayStoreFixture(t *testing.T) (*Sqlite, string) {
	t.Helper()
	s, _, path := migrationDB(t, 12)
	seedIdentityHistory(t, s)
	execSQL(t, s, "UPDATE relay_delivery_gate SET policy='durable'")
	require.NoError(t, s.SetProfileRelay(t.Context(), 1, "https://RELAY.test:443/prefix", "relay-fixture"))
	require.NoError(t, s.SetProfileRelay(t.Context(), 2, "https://else.test", "relay-fixture-two"))
	return s, path
}

// resolveRelayV9 uses the real finalizer with an already associated synthetic track.
func resolveRelayV9(t *testing.T, s *Sqlite) {
	t.Helper()
	r, _, instant, err := s.claimNextIngest(t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, r.id)
	at, err := time.Parse(receiptTime, instant)
	require.NoError(t, err)
	r.track, r.unix = 1, at.Unix()
	require.NoError(t, s.finishIngest(t.Context(), r))
}

// TestRelayOutboxClaims exercises accepted/duplicate decisions, fencing, ordering and both formats.
func TestRelayOutboxClaims(t *testing.T) {
	s, path := relayStoreFixture(t)
	ctx := t.Context()
	instant := time.Now().UTC().Truncate(time.Second)
	for i, id := range []string{"one", "duplicate", "two"} {
		stamp := instant
		if i == 2 {
			stamp = stamp.Add(time.Hour)
		}
		relayReceipt(t, s, 1, id, "v9", 1, stamp)
	}
	for range 3 {
		resolveRelayV9(t, s)
	}
	require.Equal(t, 2, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE mode='active' AND state='duplicate'"))
	execSQL(t, s, `UPDATE outbox SET format_version=2 WHERE event_id=(SELECT event_id FROM ingest_events WHERE receipt_id='two')`)
	other, err := openExisting(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, other.Close()) })
	competitor := &Sqlite{db: other}
	now := time.Now().Add(time.Minute)
	first, err := s.ClaimRelayBatch(ctx, now)
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.Equal(t, 4, first.PostBudget)
	stolen, err := competitor.ClaimRelayBatch(ctx, now)
	require.NoError(t, err)
	require.Empty(t, stolen.Items)
	for _, item := range first.Items {
		body, err := s.RelayPayload(ctx, first, item)
		require.NoError(t, err)
		require.Contains(t, string(body), "Relay Track")
		version := scalar(t, s, "SELECT format_version FROM outbox WHERE id=?", item.ID)
		if version == 1 {
			require.Contains(t, string(body), `Artist \u0026 Co`)
		} else {
			require.Contains(t, string(body), "Artist & Co")
		}
		wrong := first
		wrong.Profile = 2
		_, err = s.RelayPayload(ctx, wrong, item)
		require.ErrorIs(t, err, sql.ErrNoRows)
	}
	late := now.Add(3 * time.Minute)
	execSQL(t, s, "UPDATE relay_destinations SET pause_class='server',paused_until=1,backoff_count=3 WHERE profile_id=1")
	finished, err := s.FinishRelayBatch(ctx, first, first.Items[:1], relay.Result{Class: relay.Success, Status: 200}, late, false)
	require.NoError(t, err)
	require.True(t, finished)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_destinations WHERE profile_id=1 AND pause_class IS NULL AND paused_until IS NULL AND backoff_count=0"))
	second, err := competitor.ClaimRelayBatch(ctx, late)
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	finished, err = s.FinishRelayBatch(ctx, first, first.Items[1:], relay.Result{Class: relay.Success, Status: 200}, late, true)
	require.NoError(t, err)
	require.False(t, finished)
	renewed, err := competitor.RenewRelayBatch(ctx, second, late)
	require.NoError(t, err)
	require.True(t, renewed)
	finished, err = competitor.FinishRelayBatch(ctx, second, second.Items, relay.Result{Class: relay.Success, Status: 200, RemainingZero: true, ServerWait: time.Minute}, late, true)
	require.NoError(t, err)
	require.True(t, finished)
	require.Equal(t, 2, scalar(t, s, "SELECT count(*) FROM outbox WHERE state='sent'"))
	require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.EqualValues(t, late.Add(time.Minute).Unix(), scalar(t, s, "SELECT paused_until FROM relay_destinations WHERE profile_id=1"))
	require.Zero(t, scalar(t, s, "SELECT backoff_count FROM relay_destinations WHERE profile_id=1"))
}

// TestRelayBatchWritesUseIDs leaves an unrelated row untouched even if it has the same lease owner.
func TestRelayBatchWritesUseIDs(t *testing.T) {
	s, _ := relayStoreFixture(t)
	now := time.Now().Add(time.Minute).UTC()
	relayReceipt(t, s, 1, "first", "v9", 1, now)
	resolveRelayV9(t, s)
	batch, err := s.ClaimRelayBatch(t.Context(), now)
	require.NoError(t, err)
	require.Len(t, batch.Items, 1)
	relayReceipt(t, s, 1, "other", "v9", 1, now.Add(time.Hour))
	resolveRelayV9(t, s)
	execSQL(t, s, `UPDATE outbox SET lease_owner=?,lease_until=1 WHERE id<>?`, batch.Owner, batch.Items[0].ID)
	renewed, err := s.RenewRelayBatch(t.Context(), batch, now)
	require.NoError(t, err)
	require.True(t, renewed)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE lease_until=1"))
	_, err = s.FinishRelayBatch(t.Context(), batch, nil, relay.Result{}, now, true)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE lease_until=1"))
}

// TestRelayLeaseQueryPlans records the scan and indexed forms for renewal and release.
func TestRelayLeaseQueryPlans(t *testing.T) {
	s, _ := relayStoreFixture(t)
	cases := []struct {
		name, query string
		args        []any
	}{
		{"renew-before", `UPDATE outbox SET lease_until=? WHERE profile_id=? AND lease_owner=?`, []any{1, 1, "fixture"}},
		{"release-before", `UPDATE outbox SET state='retryable',next_attempt_at=?,lease_owner=NULL,lease_until=NULL WHERE profile_id=? AND lease_owner=? AND ` + relayDue, []any{1, 1, "fixture"}},
		{"renew-after", renewRelayItems, []any{1, `["item"]`, 1, 1, 1, "fixture"}},
		{"release-after", releaseRelayItems, []any{1, `["item"]`, 1, 1, 1, "fixture"}},
		{"expire-due", expireRelayWork + relayDue + " LIMIT 100)", []any{1, 1, 1, 1, 1}},
		{"expire-capacity", expireRelayWork + relayCapacity + " LIMIT 100)", []any{1, 1, 1, 1, 1}},
	}
	for _, tc := range cases {
		rows, err := s.db.Query("EXPLAIN QUERY PLAN "+tc.query, tc.args...)
		require.NoError(t, err)
		var steps []string
		for rows.Next() {
			var a, b, c int
			var detail string
			require.NoError(t, rows.Scan(&a, &b, &c, &detail))
			steps = append(steps, detail)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		t.Logf("%s: %s", tc.name, strings.Join(steps, " | "))
		if strings.HasSuffix(tc.name, "-after") {
			require.Contains(t, steps[0], "sqlite_autoindex_outbox_1 (id=?)")
		}
	}
}

// TestRelayPauseAndCancellation distinguishes destination failures from item rejection.
func TestRelayPauseAndCancellation(t *testing.T) {
	s, _ := relayStoreFixture(t)
	now := time.Now().Add(time.Minute).UTC().Truncate(time.Second)
	relayReceipt(t, s, 1, "one", "v9", 1, now)
	relayReceipt(t, s, 2, "two", "v9", 1, now)
	resolveRelayV9(t, s)
	resolveRelayV9(t, s)
	batch, err := s.ClaimRelayBatch(t.Context(), now)
	require.NoError(t, err)
	require.EqualValues(t, 1, batch.Profile)
	_, err = s.FinishRelayBatch(t.Context(), batch, batch.Items, relay.Result{Class: relay.Auth, Status: 401}, now, false)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_destinations WHERE profile_id=1 AND pause_class='auth' AND paused_until IS NULL"))
	next, err := s.ClaimRelayBatch(t.Context(), now)
	require.NoError(t, err)
	require.EqualValues(t, 2, next.Profile)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = s.FinishRelayBatch(ctx, next, next.Items, relay.Result{Class: relay.Transient}, now, false)
	require.NoError(t, err)
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM outbox WHERE attempt_count<>0 OR lease_owner IS NOT NULL"))
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM relay_destinations WHERE profile_id=2 AND pause_class IS NOT NULL"))
	next, err = s.ClaimRelayBatch(t.Context(), now)
	require.NoError(t, err)
	_, err = s.FinishRelayBatch(t.Context(), next, next.Items, relay.Result{Class: relay.BadRequest, Status: 413}, now, true)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE profile_id=2 AND attempt_count=1 AND state='failed'"))
	execSQL(t, s, "UPDATE outbox SET lease_owner='dead-worker',lease_until=? WHERE profile_id=1", now.Add(time.Minute).Unix())
	expired, err := s.ExpireRelayWork(t.Context(), 1, now.Add(relayMaxAge+time.Minute))
	require.NoError(t, err)
	require.EqualValues(t, 1, expired)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE profile_id=1 AND json_extract(result,'$.reason')='expired'"))
	require.Equal(t, 2, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE delivery_policy='durable' AND state='accepted'"))
}

// TestRelayOccurrenceIntent verifies the occurrence path and pending decision suppression.
func TestRelayOccurrenceIntent(t *testing.T) {
	s, _ := relayStoreFixture(t)
	execSQL(t, s, "DELETE FROM ingest_events WHERE mode='shadow'; INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
	now := time.Now().UTC().Truncate(time.Second)
	relayReceipt(t, s, 1, "accepted", "occurrence", 0, now)
	relayReceipt(t, s, 1, "ambiguous", "occurrence", 0, now.Add(time.Hour))
	// A snapshot without release cannot materialize an occurrence.
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,delivery_policy)
		VALUES('missing',1,'origin:1',1,'listenbrainz','1','missing',0,'single',?,?,'second','start',
		'{"artist":"Artist","title":"Missing release"}','fixture','1','active','occurrence','durable')`, now.Unix(), now.Add(2*time.Hour).Format(receiptTime))
	execSQL(t, s, `UPDATE ingest_events SET reason_codes='["client_batch_conflict"]' WHERE receipt_id='ambiguous'`)
	for range 3 {
		resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	}
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE receipt_id='ambiguous' AND state='pending_duplicate'"))
	require.Greater(t, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE mode='active' AND state='accepted'"), 0)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE receipt_id='missing' AND state='pending_identity'"))
	require.Equal(t, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE mode='active' AND state='accepted'"),
		scalar(t, s, "SELECT count(*) FROM outbox WHERE format_version=1 AND delivery_key=event_id"))
}

// TestRelaySourceRevisionAndMerge keeps the accepted payload stable after catalog updates.
func TestRelaySourceRevisionAndMerge(t *testing.T) {
	s, _ := relayStoreFixture(t)
	now := time.Now().Add(time.Minute).UTC().Truncate(time.Second)
	relayReceipt(t, s, 1, "source-revised", "v9", 1, now)
	resolveRelayV9(t, s)
	batch, err := s.ClaimRelayBatch(t.Context(), now)
	require.NoError(t, err)
	require.Len(t, batch.Items, 1)
	payload, err := s.RelayPayload(t.Context(), batch, batch.Items[0])
	require.NoError(t, err)
	execSQL(t, s, `INSERT INTO ingest_revisions
		SELECT profile_id,receipt_id,resolution_version+1,'user',duplicate_decision,reason_codes,evidence,rule_version,unixepoch()
		FROM ingest_events WHERE receipt_id='source-revised';
		UPDATE ingest_events SET resolution_version=resolution_version+1 WHERE receipt_id='source-revised'`)
	require.NoError(t, s.MergeTracks(t.Context(), 1, 70))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM ingest_events WHERE receipt_id='source-revised' AND occurrence_id=70 AND state='accepted'"))
	after, err := s.RelayPayload(t.Context(), batch, batch.Items[0])
	require.NoError(t, err)
	require.Equal(t, payload, after)
}

// TestRelayHistoryDeletion removes open claims and keeps only source-free sent diagnostics.
func TestRelayHistoryDeletion(t *testing.T) {
	for _, operation := range []string{"single", "profile-history", "purge"} {
		t.Run(operation, func(t *testing.T) {
			s, _ := relayStoreFixture(t)
			ctx := db.WithProfile(t.Context(), 1)
			now := time.Now().Add(time.Minute).UTC().Truncate(time.Second)
			relayReceipt(t, s, 1, "sent-source", "v9", 1, now)
			relayReceipt(t, s, 1, "open-source", "v9", 1, now.Add(time.Hour))
			resolveRelayV9(t, s)
			resolveRelayV9(t, s)
			batch, err := s.ClaimRelayBatch(ctx, now)
			require.NoError(t, err)
			require.Len(t, batch.Items, 2)
			var sent, open RelayItem
			for _, item := range batch.Items {
				if scalar(t, s, `SELECT count(*) FROM outbox o JOIN ingest_events e ON e.event_id=o.event_id
					WHERE o.id=? AND e.receipt_id='sent-source'`, item.ID) == 1 {
					sent = item
				} else {
					open = item
				}
			}
			_, err = s.FinishRelayBatch(ctx, batch, []RelayItem{sent}, relay.Result{Class: relay.Success, Status: 200}, now, false)
			require.NoError(t, err)
			switch operation {
			case "single":
				require.NoError(t, s.DeleteListen(ctx, 1, now))
				require.NoError(t, s.DeleteListen(ctx, 1, now.Add(time.Hour)))
			case "profile-history":
				plan, err := s.PreviewProfileDelete(ctx, 1, "track", 1)
				require.NoError(t, err)
				_, err = s.ApplyProfileDelete(ctx, plan)
				require.NoError(t, err)
			case "purge":
				require.NoError(t, s.PurgeAllData(ctx))
			}
			require.Zero(t, scalar(t, s, "SELECT count(*) FROM outbox"))
			require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
			require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_sent_archive WHERE id=? AND http_status=200", sent.ID))
			_, err = s.RelayPayload(ctx, batch, open)
			require.ErrorIs(t, err, sql.ErrNoRows)
			renewed, err := s.RenewRelayBatch(ctx, batch, now)
			require.NoError(t, err)
			require.False(t, renewed)
			_, err = s.FinishRelayBatch(ctx, batch, []RelayItem{open}, relay.Result{Class: relay.Success, Status: 200}, now, true)
			require.NoError(t, err)
			require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_sent_archive"))
			require.NoError(t, checkIntegrity(s.db))
		})
	}
}

// TestRelayConfigurationTransitions checks that only a real URL change loses eligibility.
func TestRelayConfigurationTransitions(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	relayReceipt(t, s, 1, "receipt", "v9", 1, time.Now())
	resolveRelayV9(t, s)
	require.NoError(t, s.SetProfileRelay(ctx, 1, "HTTPS://relay.TEST.:443/prefix/", "relay-fixture"))
	require.Equal(t, 1, scalar(t, s, "SELECT config_version FROM relay_destinations WHERE profile_id=1"))

	require.NoError(t, s.SetProfileRelay(ctx, 1, "https://relay.test/prefix", "rotated"))
	require.Equal(t, 1, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.Equal(t, 1, scalar(t, s, "SELECT generation FROM relay_destinations WHERE profile_id=1"))
	require.NoError(t, s.SetProfileRelay(ctx, 1, "https://relay.test/changed", "rotated"))
	require.Equal(t, 2, scalar(t, s, "SELECT generation FROM relay_destinations WHERE profile_id=1"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE state='pending' AND relay_generation=1"))
	require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	empty, err := s.ClaimRelayBatch(ctx, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Empty(t, empty.Items)
	_, err = s.ExpireRelayWork(ctx, 1, time.Now())
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE state='failed' AND json_extract(result,'$.reason')='destination_changed'"))
	require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.Error(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "relay.test:443", Prefix: netip.MustParsePrefix("100.64.0.0/10"), Private: true}))
	require.Error(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "relay.test:443", Prefix: netip.MustParsePrefix("::/0"), Private: true}))
	require.NoError(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "RELAY.TEST.:443", Prefix: netip.MustParsePrefix("10.0.0.1/32"), Private: true}))
	require.Error(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "relay.test:443", Prefix: netip.MustParsePrefix("8.8.8.8/32"), Private: true}))
	require.Error(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "relay.test:80", Prefix: netip.MustParsePrefix("10.0.0.1/32"), HTTP: true}))
	require.NoError(t, s.SetRelayPermission(ctx, 1, RelayPermission{HostPort: "relay.test:80", HTTP: true}))
	list, err := s.RelayPermissions(ctx, 2)
	require.NoError(t, err)
	require.Empty(t, list)
}

// TestRelayGenerationFencesClaim protects new work from a previous URL's lease.
func TestRelayGenerationFencesClaim(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	now := time.Now().Add(time.Minute).UTC()
	relayReceipt(t, s, 1, "old", "v9", 1, now)
	resolveRelayV9(t, s)
	old, err := s.ClaimRelayBatch(ctx, now)
	require.NoError(t, err)
	require.Len(t, old.Items, 1)
	require.NoError(t, s.SetProfileRelay(ctx, 1, "https://new.test/prefix", "fixture"))
	relayReceipt(t, s, 1, "new", "v9", 1, now.Add(time.Hour))
	resolveRelayV9(t, s)
	busy, err := s.ClaimRelayBatch(ctx, now)
	require.NoError(t, err)
	require.Empty(t, busy.Items)
	renewed, err := s.RenewRelayBatch(ctx, old, now)
	require.NoError(t, err)
	require.False(t, renewed)
	finished, err := s.FinishRelayBatch(ctx, old, old.Items, relay.Result{Class: relay.Success, Status: 200}, now, true)
	require.NoError(t, err)
	require.False(t, finished)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM outbox WHERE state='failed' AND json_extract(result,'$.reason')='destination_changed'`))
	require.Equal(t, 1, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	next, err := s.ClaimRelayBatch(ctx, now)
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	require.Equal(t, old.Generation+1, next.Generation)
}

// TestRelayBootstrapBinding keeps an already configured destination on boot.
func TestRelayBootstrapBinding(t *testing.T) {
	s, _, _ := migrationDB(t, 12)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	execSQL(t, s, "UPDATE users SET role='admin',status='active',password=X'01' WHERE id=1")
	require.NoError(t, s.SetProfileRelay(t.Context(), 1, "https://relay.test", "fixture"))
	require.NoError(t, s.InitializeLegacyRelay(t.Context(), "http://ignored.test", "different"))
	require.Equal(t, 1, scalar(t, s, "SELECT config_version FROM relay_destinations"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM profile_relays WHERE url='https://relay.test'"))
}

// TestRelayBootstrapSeedsAdmin covers a new install configured through the environment.
func TestRelayBootstrapSeedsAdmin(t *testing.T) {
	s, _, _ := migrationDB(t, 12)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	execSQL(t, s, "UPDATE users SET role='admin',status='active',password=X'01' WHERE id=1")
	require.NoError(t, s.InitializeLegacyRelay(t.Context(), "HTTP://127.0.0.1:80/prefix", "fixture"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE origin='seeded' AND allow_http=1 AND cidr=''"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM relay_allowlist WHERE origin='seeded' AND allow_private=1 AND cidr='127.0.0.1/32'"))
}

// TestRelayCounterCrash proves that outbox and counter share rollback and process recovery.
func TestRelayCounterCrash(t *testing.T) {
	if path := os.Getenv("TUNETRAIL_RELAY_CRASH_DB"); path != "" {
		db, err := openExisting(path)
		require.NoError(t, err)
		tx, err := db.Begin()
		require.NoError(t, err)
		_, err = tx.Exec("UPDATE outbox SET state='failed' WHERE profile_id=1")
		require.NoError(t, err)
		os.Exit(81)
	}
	s, path := relayStoreFixture(t)
	relayReceipt(t, s, 1, "receipt", "v9", 1, time.Now())
	resolveRelayV9(t, s)
	err := s.withIngestWriter(t.Context(), func(tx *sql.Conn) error {
		_, err := tx.ExecContext(t.Context(), "UPDATE outbox SET state='failed'")
		if err != nil {
			return err
		}
		return errors.New("abort fixture")
	})
	require.ErrorContains(t, err, "abort fixture")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestRelayCounterCrash$")
	cmd.Env = append(os.Environ(), "TUNETRAIL_RELAY_CRASH_DB="+path)
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	require.ErrorAs(t, err, &exit, "%s", output)
	require.Equal(t, 81, exit.ExitCode(), "%s", output)
	require.Equal(t, 1, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE state='pending'"))
}

// fillRelayCapacity creates 100,000 genuine open items.
func fillRelayCapacity(t *testing.T, s *Sqlite, now time.Time) {
	t.Helper()
	execSQL(t, s, `WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<100000)
		INSERT INTO listens(profile_id,track_id,listened_at,event_id)
		SELECT 1,1,x+1000000,printf('capacity-%06d',x) FROM n`)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state,identity_status,event_id,occurrence_id,recording_id,
		mode,delivery_policy,closed_at)
		SELECT l.event_id,1,'source',1,'listenbrainz','1',l.event_id,0,'single',?,?,'second','start',
		'{"artist":"Fixture","title":"Queue"}','fixture','1','accepted','resolved',l.event_id,1,o.recording_id,'active','durable',?
		FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=1 AND l.event_id LIKE 'capacity-%'`, now.Unix(), now.Format(receiptTime), now.Unix())
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('relay:1',1,'listenbrainz-relay',0)`)
	execSQL(t, s, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,next_attempt_at,format_version)
		SELECT event_id,1,'relay:1',event_id,'listen',event_id,?, ?,1 FROM ingest_events WHERE receipt_id LIKE 'capacity-%'`, now.Unix(), now.Unix())
}

// TestRelayMutationMeasurements times writer operations at full capacity.
func TestRelayMutationMeasurements(t *testing.T) {
	if os.Getenv("TUNETRAIL_RELAY_MEASURE") != "1" {
		t.Skip("opt-in rehearsal")
	}
	for _, operation := range []string{"url", "remove", "profile", "expire"} {
		t.Run(operation, func(t *testing.T) {
			s, _ := relayStoreFixture(t)
			fillRelayCapacity(t, s, time.Now().UTC())
			start := time.Now()
			var err error
			switch operation {
			case "url":
				err = s.SetProfileRelay(t.Context(), 1, "https://fixture.test/new", "synthetic")
			case "remove":
				err = s.SetProfileRelay(t.Context(), 1, "", "")
			case "profile":
				err = s.DeleteListeningProfile(t.Context(), 1, "One", 100002)
			case "expire":
				_, err = s.ExpireRelayWork(t.Context(), 1, time.Now())
			}
			elapsed := time.Since(start)
			require.NoError(t, err)
			t.Logf("operation=%s elapsed=%s", operation, elapsed)
			if operation == "url" || operation == "remove" {
				require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
				require.Equal(t, 100000, scalar(t, s, "SELECT count(*) FROM outbox WHERE state='pending'"))
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox o JOIN relay_destinations d USING(profile_id)
					WHERE o.profile_id=1 AND o.relay_generation=d.generation AND o.state IN ('pending','retryable')`))
			}
		})
	}
}

// TestRelayCapacity fills a real queue, then readmits the oldest overflow item after delivery.
func TestRelayCapacity(t *testing.T) {
	s, _ := relayStoreFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	fillRelayCapacity(t, s, now)
	relayReceipt(t, s, 1, "overflow", "v9", 1, now.Add(time.Hour))
	resolveRelayV9(t, s)
	require.Equal(t, 100000, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	require.Equal(t, 1, scalar(t, s, "SELECT count(*) FROM outbox WHERE json_extract(result,'$.reason')='queue_capacity'"))
	batch, err := s.ClaimRelayBatch(t.Context(), now.Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, batch.Items, 100)
	require.Equal(t, "capacity-000001", batch.Items[0].ID)
	require.Equal(t, 16, batch.PostBudget)
	_, err = s.FinishRelayBatch(t.Context(), batch, batch.Items[:1], relay.Result{Class: relay.Success, Status: 200}, now.Add(time.Minute), true)
	require.NoError(t, err)
	require.Equal(t, 99999, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
	_, err = s.ClaimRelayBatch(t.Context(), now.Add(time.Minute))
	require.NoError(t, err)
	require.Zero(t, scalar(t, s, "SELECT count(*) FROM outbox WHERE json_extract(result,'$.reason')='queue_capacity'"))
	require.Equal(t, scalar(t, s, "SELECT count(*) FROM outbox WHERE state IN ('pending','retryable')"), scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
}

// TestRelayMigrationMeasurements reports an opt-in 300,000 receipt rehearsal.
func TestRelayMigrationMeasurements(t *testing.T) {
	if os.Getenv("TUNETRAIL_RELAY_MEASURE") != "1" {
		t.Skip("opt-in rehearsal")
	}
	s, _, path := migrationDB(t, 11)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	execSQL(t, s, "INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0)")
	execSQL(t, s, `WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<300000)
	INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
	received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version)
	SELECT printf('receipt-%06d',x),1,'source',1,'test','1','batch',x,'single',x,'second','start',
	'{"artist":"Fixture Artist","title":"Fixture Title","release":"Fixture Album"}','fixture','1' FROM n`)
	execSQL(t, s, "PRAGMA wal_checkpoint(TRUNCATE)")
	before, err := os.Stat(path)
	require.NoError(t, err)
	start := time.Now()
	backup, err := backupBeforeUpgrade(t.Context(), s.db, path, 11)
	require.NoError(t, err)
	backupTime := time.Since(start)
	start = time.Now()
	require.NoError(t, upgradeRelaySchema(t.Context(), s.db, lookupFixture{}))
	migrationTime := time.Since(start)
	wal, err := os.Stat(path + "-wal")
	require.NoError(t, err)
	saved, err := os.Stat(backup)
	require.NoError(t, err)
	t.Logf("recibos=300000 banco=%d backup=%d tempo_backup=%s tempo_012=%s wal_012=%d", before.Size(), saved.Size(), backupTime, migrationTime, wal.Size())
	require.True(t, strings.Contains(filepath.Base(backup), ".pre-relay-delivery-"))
	// Measure the CHECK scan alone on the disposable v11 backup after recording its size.
	v11, err := openExisting(backup)
	require.NoError(t, err)
	defer v11.Close()
	require.Equal(t, 11, scalar(t, &Sqlite{db: v11}, "SELECT max(version_id) FROM goose_db_version"))
	require.Equal(t, 300000, scalar(t, &Sqlite{db: v11}, "SELECT count(*) FROM ingest_events"))
	_, err = v11.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_checkpoint(TRUNCATE)")
	require.NoError(t, err)
	start = time.Now()
	_, err = v11.Exec("ALTER TABLE ingest_events ADD COLUMN delivery_policy TEXT NOT NULL DEFAULT 'legacy' CHECK(delivery_policy IN ('legacy','durable'))")
	require.NoError(t, err)
	checkTime := time.Since(start)
	checkWAL, err := os.Stat(backup + "-wal")
	require.NoError(t, err)
	t.Logf("check_isolado=%s wal_check_isolado=%d", checkTime, checkWAL.Size())

	for _, query := range []struct {
		sql  string
		args []any
	}{
		{relayCandidate, []any{1, 1, 1}},
		{`SELECT id FROM outbox WHERE profile_id=? AND destination_id='relay:'||? AND relay_generation=? AND ` + relayDue + ` AND next_attempt_at<=? AND coalesce(lease_until,0)<=? AND created_at>? ORDER BY next_attempt_at,id LIMIT 100`, []any{1, 1, 1, 1, 1, 0}},
	} {
		rows, err := s.db.Query("EXPLAIN QUERY PLAN "+query.sql, query.args...)
		require.NoError(t, err)
		for rows.Next() {
			var a, b, c int
			var description string
			require.NoError(t, rows.Scan(&a, &b, &c, &description))
			t.Log(description)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	require.NoError(t, checkDatabase(s.db))
}

// TestRelayMarkerMeasurements isolates schema costs on a populated legacy outbox.
func TestRelayMarkerMeasurements(t *testing.T) {
	if os.Getenv("TUNETRAIL_RELAY_MEASURE") != "1" {
		t.Skip("opt-in rehearsal")
	}
	s, _, path := migrationDB(t, 11)
	seedIdentityHistory(t, s)
	execSQL(t, s, `WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<300000)
		INSERT INTO listens(profile_id,track_id,listened_at,event_id)
		SELECT 1,1,x+1000000,printf('marker-%06d',x) FROM n`)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state,identity_status,event_id,occurrence_id,recording_id,mode,closed_at)
		SELECT l.event_id,1,'source',1,'listenbrainz','1',l.event_id,0,'single',1,?,'second','start',
		'{"artist":"Fixture","title":"Queue"}','fixture','1','accepted','resolved',l.event_id,1,o.recording_id,'active',1
		FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=1 AND l.event_id LIKE 'marker-%'`, time.Unix(1, 0).UTC().Format(receiptTime))
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('relay:1',1,'listenbrainz-relay',0);
		INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,closed_at,result)
		SELECT event_id,1,'relay:1',event_id,'listen',event_id,'discarded',1,1,'{"reason":"pre_082_best_effort"}'
		FROM ingest_events WHERE receipt_id LIKE 'marker-%'`)
	execSQL(t, s, "PRAGMA wal_checkpoint(TRUNCATE)")
	start := time.Now()
	backup, err := backupBeforeUpgrade(t.Context(), s.db, path, 11)
	require.NoError(t, err)
	backupElapsed := time.Since(start)
	size, err := os.Stat(backup)
	require.NoError(t, err)
	start = time.Now()
	require.NoError(t, upgradeRelaySchema(t.Context(), s.db, lookupFixture{}))
	upgradeElapsed := time.Since(start)
	wal, err := os.Stat(path + "-wal")
	require.NoError(t, err)
	require.Equal(t, 300000, scalar(t, s, `SELECT count(*) FROM outbox WHERE format_version=0 AND state='discarded' AND json_extract(result,'$.reason')='pre_082_best_effort'`))
	t.Logf("marcadores=300000 backup=%s backup_bytes=%d upgrade=%s wal_012=%d", backupElapsed, size.Size(), upgradeElapsed, wal.Size())

	copyDB, err := openExisting(backup)
	require.NoError(t, err)
	defer copyDB.Close()
	_, err = copyDB.Exec("PRAGMA journal_mode=WAL; PRAGMA wal_checkpoint(TRUNCATE)")
	require.NoError(t, err)
	start = time.Now()
	_, err = copyDB.Exec("ALTER TABLE outbox ADD COLUMN format_version INTEGER NOT NULL DEFAULT 0 CHECK(format_version>=0)")
	require.NoError(t, err)
	formatElapsed := time.Since(start)
	formatWAL, err := os.Stat(backup + "-wal")
	require.NoError(t, err)
	t.Logf("check_format=%s wal_format=%d", formatElapsed, formatWAL.Size())
	_, err = copyDB.Exec("ALTER TABLE outbox ADD COLUMN relay_generation INTEGER NOT NULL DEFAULT 1 CHECK(relay_generation>0); PRAGMA wal_checkpoint(TRUNCATE)")
	require.NoError(t, err)
	start = time.Now()
	_, err = copyDB.Exec(`CREATE INDEX idx_outbox_relay_due ON outbox(profile_id,destination_id,relay_generation,next_attempt_at,id)
		WHERE format_version>0 AND state IN ('pending','retryable');
		CREATE INDEX idx_outbox_relay_capacity ON outbox(profile_id,destination_id,relay_generation,created_at,id)
		WHERE format_version>0 AND state='failed' AND json_extract(result,'$.reason')='queue_capacity'`)
	require.NoError(t, err)
	indexElapsed := time.Since(start)
	indexWAL, err := os.Stat(backup + "-wal")
	require.NoError(t, err)
	t.Logf("indices_parciais=%s wal_indices=%d", indexElapsed, indexWAL.Size())
}
