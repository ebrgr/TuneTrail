package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestTrackRankingsByIdentity covers shared recordings, release buckets and profile-local discovery.
func TestTrackRankingsByIdentity(t *testing.T) {
	s := mergeFixtureAtVersion(t, 20)
	execSQL(t, s, `
		DELETE FROM listens;
		INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('source',1,'test',0);
		UPDATE tracks SET musicbrainz_id='21dce41d-297e-44ac-a3ed-73365d822185' WHERE id=1;
		UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=1) WHERE id=2;
		INSERT INTO tracks(id,release_id) VALUES(3,1),(4,1),(5,1);
		INSERT INTO track_aliases VALUES(3,'Same title','User',1),(4,'Same title','User',1),(5,'One','User',1);
		UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=1) WHERE id=5;
		INSERT INTO occurrences(id,recording_id,presented_title)
			SELECT 6,recording_id,'Unreleased take' FROM occurrences WHERE id=1;
		INSERT INTO occurrences(id,recording_id,presented_title)
			SELECT 7,recording_id,'Another unreleased take' FROM occurrences WHERE id=1;
		INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES
			(1,1,100,'legacy'),(2,1,100,'legacy'),(1,2,200,'legacy'),
			(1,3,200,'legacy'),(1,4,200,'legacy'),(1,5,200,'legacy'),
			(1,6,200,'listenbrainz_api'),(1,7,200,'listenbrainz_api');
	`)
	insertReceipt(t, s, "pending")
	execSQL(t, s, `UPDATE ingest_events SET state='pending_duplicate', listened_at='200' WHERE receipt_id='pending'`)
	ctx := context.Background()
	alpha, beta := db.WithProfile(ctx, 1), db.WithProfile(ctx, 2)
	all := db.Timeframe{Period: db.PeriodAllTime}
	rank := func(ctx context.Context, opts db.GetItemsOpts) map[int32]int64 {
		t.Helper()
		opts.Page, opts.Limit, opts.Timeframe = 1, 100, all
		r, err := s.GetTopTracksPaginated(ctx, opts)
		require.NoError(t, err)
		counts := make(map[int32]int64)
		for _, v := range r.Items {
			counts[v.Item.ID] = v.Item.ListenCount
			if v.Item.ID == 6 {
				require.Zero(t, v.Item.AlbumID)
				require.Equal(t, "Unreleased take", v.Item.Title)
			}
		}
		require.EqualValues(t, len(counts), r.TotalCount)
		return counts
	}

	require.Equal(t, map[int32]int64{1: 6, 3: 1, 4: 1}, rank(ctx, db.GetItemsOpts{}))
	require.Equal(t, map[int32]int64{1: 5, 3: 1, 4: 1}, rank(alpha, db.GetItemsOpts{}))
	require.Equal(t, map[int32]int64{1: 1}, rank(beta, db.GetItemsOpts{}))
	release := db.GetItemsOpts{TrackGrouping: db.TrackByRelease}
	require.Equal(t, map[int32]int64{1: 3, 2: 1, 3: 1, 4: 1, 6: 2}, rank(ctx, release))
	require.Equal(t, map[int32]int64{1: 2, 2: 1, 3: 1, 4: 1, 6: 2}, rank(alpha, release))
	require.Equal(t, map[int32]int64{1: 1}, rank(beta, release))
	require.Equal(t, map[int32]int64{2: 1}, rank(alpha, db.GetItemsOpts{AlbumID: 2}))
	require.Equal(t, map[int32]int64{}, rank(beta, db.GetItemsOpts{AlbumID: 2}))
	require.Equal(t, map[int32]int64{1: 1}, rank(alpha, db.GetItemsOpts{ArtistID: 1}))

	period := db.Timeframe{FromUnix: 150, ToUnix: 250}
	heardTracks, err := s.CountTracks(alpha, period)
	require.NoError(t, err)
	require.EqualValues(t, 3, heardTracks)
	newTracks, err := s.CountNewTracks(alpha, period)
	require.NoError(t, err)
	require.EqualValues(t, 2, newTracks)
	newAlbums, err := s.CountNewAlbums(alpha, period)
	require.NoError(t, err)
	require.EqualValues(t, 1, newAlbums)
	newArtists, err := s.CountNewArtists(alpha, period)
	require.NoError(t, err)
	require.EqualValues(t, 1, newArtists)
	for _, count := range []func(context.Context, db.Timeframe) (int64, error){s.CountNewTracks, s.CountNewAlbums, s.CountNewArtists} {
		n, err := count(beta, period)
		require.NoError(t, err)
		require.Zero(t, n)
	}
}

// TestTrackRankingQueryPlans measures the production SQL against 600,000 listening events.
func TestTrackRankingQueryPlans(t *testing.T) {
	if !testing.Verbose() || testing.Short() {
		t.Skip("run explicitly with -v to measure and print query plans")
	}
	s := mergeFixture(t)
	execSQL(t, s, `
		WITH RECURSIVE seq(n) AS (VALUES(3) UNION ALL SELECT n+1 FROM seq WHERE n<10002)
		INSERT INTO tracks(id,release_id) SELECT n, 1+n%2 FROM seq;
		INSERT INTO track_aliases SELECT id,'Fixture track '||id,'User',1 FROM tracks WHERE id>=3;
		INSERT INTO artist_tracks SELECT release_id,id,1 FROM tracks WHERE id>=3;
		UPDATE occurrences AS o SET recording_id=(SELECT first.recording_id FROM occurrences first WHERE first.id=o.id-1)
			WHERE o.id>=4 AND o.id%2=0;
		WITH RECURSIVE seq(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM seq WHERE n<600000)
		INSERT INTO listens(profile_id,track_id,listened_at)
		SELECT CASE WHEN (n/10000+n)%5=0 THEN 2 ELSE 1 END, 3+((n*7919)%10000), 100000+n/2 FROM seq;
		ANALYZE;
	`)
	for _, scenario := range []struct {
		name     string
		filter   db.GetItemsOpts
		from, to int64
	}{
		{"all-time", db.GetItemsOpts{}, 1, 400001},
		{"period", db.GetItemsOpts{}, 150000, 250000},
		{"album", db.GetItemsOpts{AlbumID: 1}, 150000, 250000},
		{"artist", db.GetItemsOpts{ArtistID: 1}, 150000, 250000},
	} {
		for _, grouping := range []string{db.TrackByRecording, db.TrackByRelease} {
			t.Run(scenario.name+"/"+grouping, func(t *testing.T) {
				ctx := db.WithProfile(context.Background(), 1)
				opts := scenario.filter
				opts.TrackGrouping, opts.Page, opts.Limit = grouping, 1, 100
				opts.Timeframe = db.Timeframe{FromUnix: scenario.from, ToUnix: scenario.to}
				query, args := topTracksQuery(opts, opts.Timeframe.FromUnix, opts.Timeframe.ToUnix, 0)
				rows, err := s.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+scopedRead(ctx, query), args...)
				require.NoError(t, err)
				defer rows.Close()
				for rows.Next() {
					var id, parent, unused int
					var detail string
					require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
					t.Log(detail)
				}
				require.NoError(t, rows.Err())
				require.NoError(t, rows.Close())
				start := time.Now()
				for i := 0; i < 3; i++ {
					page, err := s.GetTopTracksPaginated(ctx, opts)
					require.NoError(t, err)
					require.Len(t, page.Items, 100)
				}
				t.Logf("mean over three complete pages: %s", time.Since(start)/3)
			})
		}
	}
}
