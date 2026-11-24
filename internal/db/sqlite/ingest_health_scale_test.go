package sqlite

import (
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestIngestHealthQueryPlans measures the production projections on a disposable database at the current schema.
// Run with TUNETRAIL_TEST_HEALTH_SCALE=1 go test ./internal/db/sqlite -run '^TestIngestHealthQueryPlans$' -count=1 -v.
func TestIngestHealthQueryPlans(t *testing.T) {
	if os.Getenv("TUNETRAIL_TEST_HEALTH_SCALE") != "1" {
		t.Skip("set TUNETRAIL_TEST_HEALTH_SCALE=1 to build a 300,000-receipt fixture")
	}
	s, _, _ := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, s, 1, "health-one", "One", 0)
	testfixtures.Profile(t, s, 2, "health-two", "Two", 0)
	seedMusic(t, s)
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at)
		VALUES('scale:1',1,'listenbrainz',0),('scale:2',1,'listenbrainz',0),('scale:3',2,'listenbrainz',0);
		WITH RECURSIVE catalog(n) AS (VALUES(101) UNION ALL SELECT n+1 FROM catalog WHERE n<5100)
		INSERT INTO tracks(id,release_id,duration) SELECT n,1,180 FROM catalog;
		INSERT INTO track_aliases(track_id,alias,source,is_primary)
		SELECT id,'Example track '||id,'fixture',1 FROM tracks WHERE id BETWEEN 101 AND 5100;
		WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<300000)
		INSERT INTO listens(profile_id,track_id,listened_at,event_id,origin)
		SELECT CASE WHEN n%6=0 THEN 2 ELSE 1 END,101+(n%5000),1700000000+n*8,
			printf('event:%07d',n),'listenbrainz_api' FROM sample;
		WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<300000)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
			mode,resolver_variant,state,identity_status,duplicate_decision,resolved_at,closed_at,event_id,occurrence_id,recording_id,evidence)
		SELECT printf('scale:%07d',n),l.profile_id,
			CASE WHEN n%6=0 THEN 'scale:3' WHEN n%3=0 THEN 'scale:2' ELSE 'scale:1' END,
			1,'listenbrainz','1','scale',n,'single',l.listened_at,
			strftime('%Y-%m-%dT%H:%M:%S.000000000Z',l.listened_at,'unixepoch'),'second','start',
			json_object('title','Example title','artist','Example artist','release','Example release','client','player-'||(n%12)),
			'fixture','1','active','occurrence',CASE WHEN n%997=0 THEN 'pending_duplicate' ELSE 'accepted' END,
			'resolved',CASE WHEN n%997=0 THEN 'review' ELSE 'different' END,
			l.listened_at+1,CASE WHEN n%997<>0 THEN l.listened_at+1 END,
			CASE WHEN n%997<>0 THEN l.event_id END,o.id,o.recording_id,
			CASE WHEN n%997=0 THEN json_object('candidates',json_array(l.event_id)) ELSE '{}' END
		FROM sample JOIN listens l ON l.event_id=printf('event:%07d',n) JOIN occurrences o ON o.id=l.track_id;
		ANALYZE`)
	require.Equal(t, 250000, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1`))
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
	t.Logf("schema v%d", currentSchemaVersion)
	sources := healthSourcesSQL()
	explain("sources", sources, 1, 0)
	explain("decisions", healthDecisionsSQL, 1)
	explain("pending list", reviewPageSQL, 1, reviewPageSize, 0)
	explain("pending count", reviewCountSQL, 1)
	explain("candidates", reviewCandidateRows, 1, "scale:0298103")
	now := time.Unix(1700000000+300000*8, 0)
	for _, days := range []int{1, 7, 30} {
		since := now.Add(-time.Duration(days) * 24 * time.Hour)
		name := fmt.Sprintf("Inbox %dd", days)
		explain(name, inboxCountsSQL, inboxCountArgs(1, -1, int64(1<<63-1), since, now)...)
		measure(name, func() error {
			_, err := s.IngestInbox(t.Context(), 1, nil, since, now)
			return err
		})
	}
	measure("sources", func() error {
		_, err := s.healthSources(t.Context(), 1, 0)
		return err
	})
	measure("decisions", func() error {
		_, err := s.healthDecisions(t.Context(), 1)
		return err
	})
	var health db.IngestHealth
	measure("pending count + page + candidates", func() error {
		health = db.IngestHealth{}
		return s.healthReviews(t.Context(), 1, 1, &health)
	})
	require.Equal(t, 250, health.ReviewCount)
	require.Len(t, health.Reviews, reviewPageSize)
}
