package sqlite

import (
	"os"
	"slices"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestIngestHealthQueryPlans measures the production projections on a disposable v15-to-v17 database.
// Run with TUNETRAIL_TEST_HEALTH_SCALE=1 go test ./internal/db/sqlite -run '^TestIngestHealthQueryPlans$' -count=1 -v.
func TestIngestHealthQueryPlans(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_HEALTH_SCALE") != "1" {
		t.Skip("set TUNETRAIL_TEST_HEALTH_SCALE=1 to build a 300,000-receipt fixture")
	}
	s, provider, _ := migrationDB(t, 15)
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at)
		VALUES('scale:1',1,'listenbrainz',0),('scale:2',1,'file',0),('scale:3',2,'listenbrainz',0);
		WITH RECURSIVE catalog(n) AS (VALUES(101) UNION ALL SELECT n+1 FROM catalog WHERE n<5100)
		INSERT INTO tracks(id,release_id,duration) SELECT n,1,180 FROM catalog;
		INSERT INTO track_aliases(track_id,alias,source,is_primary)
		SELECT id,'Example track '||id,'fixture',1 FROM tracks WHERE id BETWEEN 101 AND 5100;
		WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<300000)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,state,duplicate_decision,resolved_at,closed_at,occurrence_id,recording_id,evidence)
		SELECT printf('scale:%07d',n),CASE WHEN n%6=0 THEN 2 ELSE 1 END,
			CASE WHEN n%6=0 THEN 'scale:3' WHEN n%3=0 THEN 'scale:2' ELSE 'scale:1' END,
			1,CASE WHEN n%6=3 THEN 'file' ELSE 'listenbrainz' END,'1','scale',n,'single',1700000000+n*180,
			'2024-05-01T12:00:00.000000000Z','second','start',
			json_object('title','Example title','artist','Example artist','release','Example release','client','player-'||(n%12)),
			'fixture','1',CASE WHEN n%997=0 THEN 'active' ELSE 'shadow' END,
			CASE WHEN n%997=0 THEN 'pending_duplicate' ELSE 'received' END,
			CASE WHEN n%997=0 THEN 'review' ELSE 'different' END,
			1700000000+n*180+1,NULL,CASE WHEN n%997=0 THEN 1 END,CASE WHEN n%997=0 THEN (SELECT recording_id FROM occurrences WHERE id=1) END,
			json_object('candidate_event_id',(SELECT event_id FROM listens WHERE rowid=10))
		FROM sample;
		WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<300000)
		INSERT INTO listens(profile_id,track_id,listened_at)
		SELECT CASE WHEN n%6=0 THEN 2 ELSE 1 END,101+(n%5000),1700000000+n*180 FROM sample;
		ANALYZE`)
	require.Equal(t, 250000, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1 AND receipt_id LIKE 'scale:%'`))
	require.Equal(t, 250, scalar(t, s, reviewCountSQL, 1))

	explain := func(name, query string, args ...any) {
		t.Helper()
		rows, err := s.db.QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+query, args...)
		require.NoError(t, err)
		defer rows.Close()
		for rows.Next() {
			var id, parent, unused int
			var detail string
			require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
			t.Logf("%s plan: %s", name, detail)
		}
		require.NoError(t, rows.Err())
	}
	measure := func(name string, f func() error) {
		t.Helper()
		var samples []time.Duration
		for range 6 {
			started := time.Now()
			require.NoError(t, f())
			samples = append(samples, time.Since(started))
		}
		samples = samples[1:]
		slices.Sort(samples)
		t.Logf("%s: warm median %s (range %s–%s)", name, samples[2], samples[0], samples[4])
	}
	for _, version := range []int{15, 17} {
		if version > 15 {
			started := time.Now()
			_, err := provider.UpTo(t.Context(), int64(version))
			require.NoError(t, err)
			t.Logf("migrate to v%d: %s", version, time.Since(started))
			execSQL(t, s, `ANALYZE`)
		}
		t.Logf("schema v%d", version)
		explain("sources", healthSourcesSQL(version), 1, 0)
		explain("decisions", healthDecisionsSQL, 1)
		explain("pending list", reviewPageSQL, 1, reviewPageSize, 0)
		explain("pending count", reviewCountSQL, 1)
		explain("candidates", reviewCandidateRows, 1, "scale:0298103")
		measure("sources", func() error {
			_, err := s.healthSources(t.Context(), 1, version, 0)
			return err
		})
		measure("decisions", func() error {
			_, err := s.healthDecisions(t.Context(), 1)
			return err
		})
		measure("pending count + page + candidates", func() error {
			var health db.IngestHealth
			return s.healthReviews(t.Context(), 1, 1, &health)
		})
	}
}
