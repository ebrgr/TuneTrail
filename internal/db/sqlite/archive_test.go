package sqlite

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceArchiveRoundTrip exercises identity, catalog edits, and profile isolation.
func TestOccurrenceArchiveRoundTrip(t *testing.T) {
	ctx := t.Context()
	source := mergeFixture(t)
	execSQL(t, source, `INSERT INTO track_aliases VALUES(1,'Edited','User',0);
		INSERT INTO recordings DEFAULT VALUES;
		INSERT INTO occurrences(id,recording_id,presented_title) VALUES(90,last_insert_rowid(),'Unreleased');
		INSERT INTO occurrence_credits VALUES(90,0,'Credit',NULL);
		INSERT INTO listens(profile_id,track_id,listened_at,event_id,origin,client)
		VALUES(1,1,100,'second-event','listenbrainz_api','second client'),
		      (1,90,100,'no-release','spotify_import','Spotify')`)
	require.NoError(t, source.SetPrimaryTrackAlias(ctx, 1, "Edited"))
	require.Equal(t, 1, scalar(t, source, `SELECT count(*) FROM track_aliases WHERE track_id=2 AND alias='Two' AND is_primary=1`))
	require.NoError(t, source.MergeTracks(ctx, 1, 2))
	require.Equal(t, 6, scalar(t, source, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, source, `SELECT count(DISTINCT recording_id) FROM occurrences WHERE id IN (1,2)`))
	require.NoError(t, source.UndoTrackMerge(ctx, 1, 2))
	require.Equal(t, 2, scalar(t, source, `SELECT count(DISTINCT recording_id) FROM occurrences WHERE id IN (1,2)`))
	require.NoError(t, source.MergeTracks(ctx, 1, 2))

	target, _, _ := migrationDB(t, currentSchemaVersion)
	for _, profile := range []int32{1, 2} {
		testfixtures.Profile(t, target, int(profile), fmt.Sprintf("p-%d", profile), "Restored", 1)
		var items []*db.ExportItem
		cursor := ""
		var ceiling int64
		for {
			page, err := source.GetExportPage(ctx, db.GetExportPageOpts{ProfileID: profile, AfterEventID: cursor, ThroughSequence: ceiling, Limit: 1})
			require.NoError(t, err)
			if len(page) == 0 {
				break
			}
			require.Greater(t, page[0].Cursor, cursor)
			cursor = page[0].Cursor
			ceiling = page[0].ThroughSequence
			items = append(items, page...)
		}
		count, err := target.RestoreArchive(ctx, profile, items)
		require.NoError(t, err)
		require.Equal(t, len(items), count)
		count, err = target.RestoreArchive(ctx, profile, items)
		require.NoError(t, err)
		require.Zero(t, count)
		got, err := target.GetExportPage(ctx, db.GetExportPageOpts{ProfileID: profile, Limit: 20})
		require.NoError(t, err)
		require.Len(t, got, len(items))
		expected := map[string]*db.ExportItem{}
		for _, item := range items {
			expected[item.EventID] = item
		}
		for _, item := range got {
			before := expected[item.EventID]
			require.NotNil(t, before)
			require.Equal(t, before.EventNamespace, item.EventNamespace)
			require.Equal(t, before.Origin, item.Origin)
			require.Equal(t, before.Provenance, item.Provenance)
			require.Equal(t, before.ListenedAt, item.ListenedAt)
			require.Equal(t, *before.Client, *item.Client)
			require.Equal(t, before.ReleaseID == 0, item.ReleaseID == 0)
			require.Equal(t, before.TrackAliases[0].Alias, item.TrackAliases[0].Alias)
			require.Equal(t, before.TrackAliases[0].Primary, item.TrackAliases[0].Primary)
		}
	}
	require.Equal(t, 2, scalar(t, target, `SELECT count(*) FROM releases`))
	require.Equal(t, 2, scalar(t, target, `SELECT count(*) FROM artist_releases`))

	require.Equal(t, 6, scalar(t, target, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, target, `SELECT count(*) FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE o.release_id IS NULL`))

	first := db.WithProfile(ctx, 1)
	require.ErrorIs(t, source.DeleteListen(first, 1, time.Unix(100, 0)), db.ErrConflict)
	require.ErrorIs(t, source.DeleteListenEvent(db.WithProfile(ctx, 2), "second-event"), db.ErrNotFound)
	require.NoError(t, source.DeleteListenEvent(first, "second-event"))
	require.NoError(t, source.DeleteTrack(ctx, 1))
	require.Equal(t, 2, scalar(t, source, `SELECT count(*) FROM listens WHERE track_id=2 AND listened_at=100`))
	require.NoError(t, source.DeleteTrack(ctx, 90))
	require.Equal(t, 2, scalar(t, source, `SELECT count(*) FROM listens`))
	assertForeignKeys(t, source)
	assertForeignKeys(t, target)
}

// TestOccurrenceExportPlan records first and deep pages on a 300,000-event profile fixture.
func TestOccurrenceExportPlan(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT 1+n FROM seq WHERE n<300000)
		INSERT INTO listens(profile_id,track_id,listened_at,event_id,origin)
		SELECT 1+n%2,1,100,printf('event-%09d',n),'listenbrainz_api' FROM seq;
		ANALYZE`)
	for _, cursor := range []string{"", "event-000290000"} {
		rows, err := s.db.QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+exportPageQuery, 1, cursor, 300004, 1000)
		require.NoError(t, err)
		var plan []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
			plan = append(plan, detail)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		joined := strings.Join(plan, "\n")
		require.Contains(t, joined, "profile_id=? AND event_id>?")
		require.NotContains(t, joined, "TEMP B-TREE")
		t.Logf("cursor %q\n%s", cursor, joined)
		start := time.Now()
		page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, AfterEventID: cursor, ThroughSequence: 300004, Limit: 1000})
		require.NoError(t, err)
		require.Len(t, page, 1000)
		t.Logf("page with aliases and credits: %s", time.Since(start))
	}
}

// TestOccurrenceReconciliationRevisesActiveReceipts verifies the writer guard and receipt links.
func TestOccurrenceReconciliationRevisesActiveReceipts(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	ctx := t.Context()
	// The occurrence writer creates independently identified releases with provisional MBIDs.
	first := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Shared", Artist: "Artist", Release: "Edition A"}, 1700000000, "edition-a")
	resolveOccurrenceFixture(t, s, nil)
	second := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Shared", Artist: "Artist", Release: "Edition B"}, 1700000000, "edition-b")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, first, "accepted")
	requireReceiptState(t, s, second, "accepted")
	var from, to int32
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id,event_id FROM ingest_events WHERE receipt_id=?`, first).Scan(&from, &event))
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id FROM ingest_events WHERE receipt_id=?`, second).Scan(&to))
	_, err := s.db.Exec(`UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=?) WHERE id=?`, to, from)
	require.ErrorContains(t, err, "Cannot reassign")
	require.NoError(t, s.MergeTracks(ctx, from, to))
	require.Equal(t, 1, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM ingest_events WHERE receipt_id IN (?,?)`, first, second))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND occurrence_id=? AND event_id=? AND resolution_version=2`, first, from, event))
	require.NoError(t, s.UndoTrackMerge(ctx, from, to))
	require.Equal(t, 2, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM ingest_events WHERE receipt_id IN (?,?)`, first, second))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND occurrence_id=? AND event_id=? AND resolution_version=3`, first, from, event))
	page, err := s.GetExportPage(ctx, db.GetExportPageOpts{ProfileID: 1, Limit: 20})
	require.NoError(t, err)
	for _, item := range page {
		if item.EventID == event {
			require.Contains(t, string(item.Provenance), first)
			require.Contains(t, string(item.Provenance), `"resolution_version":3`)
			return
		}
	}
	t.Fatal("linked event missing from export")
}
