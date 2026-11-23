package sqlite

import (
	"go/build"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestRelayE4Store measures the maintenance tick on the combined large fixture.
func TestRelayE4Store(t *testing.T) {
	if os.Getenv("TUNETRAIL_RELAY_MEASURE") != "1" {
		t.Skip("opt-in rehearsal")
	}
	t.Logf("platform=%s/%s", build.Default.GOOS, build.Default.GOARCH)
	s, _ := relayStoreFixture(t)
	now := time.Now()
	fillRelayCapacity(t, s, now)
	execSQL(t, s, `WITH RECURSIVE n(x) AS (VALUES(1) UNION ALL SELECT x+1 FROM n WHERE x<200000)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,delivery_policy,state,closed_at,rule_version,reason_codes)
		SELECT printf('archive-%06d',x),1,'source',1,'listenbrainz','1','archive',x,'import',1,'second','start',
		'{"artist":"Fixture","title":"Archive"}','fixture','1','active','occurrence','durable','invalid',1,'fixture','["fixture"]' FROM n`)
	require.Equal(t, 300000, scalar(t, s, "SELECT count(*) FROM ingest_events"))
	require.Equal(t, 100000, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))

	for _, scope := range []string{relayDue, relayCapacity} {
		rows, err := s.db.Query("EXPLAIN QUERY PLAN "+expireRelayWork+scope+" LIMIT 100)", 1, 1, 1, now.Add(-relayMaxAge).Unix(), now.Unix())
		require.NoError(t, err)
		for rows.Next() {
			var a, b, c int
			var detail string
			require.NoError(t, rows.Scan(&a, &b, &c, &detail))
			t.Log(detail)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	for range 7 {
		begin := time.Now()
		count, err := s.ExpireRelayWork(t.Context(), 1, now)
		t.Logf("expire=%s", time.Since(begin))
		require.NoError(t, err)
		require.Zero(t, count)
	}

	rows, err := s.db.Query(`SELECT name,sum(pgsize) FROM dbstat WHERE name='outbox' OR name IN
		(SELECT name FROM sqlite_schema WHERE tbl_name='outbox' AND type='index') GROUP BY name ORDER BY name`)
	if err != nil {
		t.Logf("dbstat unavailable: %s", err)
	} else {
		for rows.Next() {
			var name string
			var bytes int64
			require.NoError(t, rows.Scan(&name, &bytes))
			t.Logf("pages %s=%d", name, bytes)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
	}
	pageSize := scalar(t, s, "PRAGMA page_size")
	filled := scalar(t, s, "PRAGMA page_count") - scalar(t, s, "PRAGMA freelist_count")
	execSQL(t, s, "SAVEPOINT measure_pages; DELETE FROM outbox")
	empty := scalar(t, s, "PRAGMA page_count") - scalar(t, s, "PRAGMA freelist_count")
	execSQL(t, s, "ROLLBACK TO measure_pages; RELEASE measure_pages")
	t.Logf("outbox_bytes_reclaimed=%d", (filled-empty)*pageSize)

	begin := time.Now()
	plan, err := s.PreviewProfileDelete(t.Context(), 1, "track", 1)
	require.NoError(t, err)
	_, err = s.ApplyProfileDelete(t.Context(), plan)
	t.Logf("history_delete_preview_apply=%s", time.Since(begin))
	require.NoError(t, err)
	require.Zero(t, scalar(t, s, "SELECT open_count FROM relay_destinations WHERE profile_id=1"))
}
