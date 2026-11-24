package sqlite

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

// TestEnrichmentReceivedArtistProjection checks the v25 backfill, live writes,
// precedence by credit, and receipt deletion on the indexed lookup.
func TestEnrichmentReceivedArtistProjection(t *testing.T) {
	s, migration, _ := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE mode='shadow' AND event_id IS NULL`)
	_, err := migration.UpTo(t.Context(), 25)
	require.NoError(t, err)
	require.NoError(t, s.SetIngestReview(t.Context(), 1, false))
	duet := ingest.Snapshot{Title: "Projected duet", Artist: "A & B", ArtistNames: []string{"A", "B"}, ArtistMbidMappings: []ingest.ArtistMapping{{Name: "A", MBID: knownArtist174}}}
	old := admitOccurrenceFixture(t, s, duet, 1700000100, "projection-old")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	_, err = migration.UpTo(t.Context(), currentSchemaVersion)
	require.NoError(t, err)
	// Standalone credits without catalog IDs remain unlinked. Give this fixture two linked credits.
	for i, name := range []string{"A", "B"} {
		artist := 500 + i
		execSQL(t, s, `INSERT INTO artists(id) VALUES (?)`, artist)
		execSQL(t, s, `UPDATE occurrence_credits SET artist_id=? WHERE name=?
			AND occurrence_id=(SELECT occurrence_id FROM ingest_events WHERE receipt_id=?)`, artist, name, old)
		execSQL(t, s, `INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES(1,?,(SELECT event_id FROM ingest_events WHERE receipt_id=?),'test',?,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid')`, artist, old, enrichedArtist174)
	}
	check := func(artist int, suppressed bool) {
		t.Helper()
		var got string
		err := s.db.QueryRow(scopedArtistClaim, 1, artist).Scan(&got)
		if suppressed {
			require.ErrorIs(t, err, sql.ErrNoRows)
		} else {
			require.NoError(t, err)
			require.Equal(t, enrichedArtist174, got)
		}
	}
	check(500, true)
	check(501, false)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_received_artist_credits WHERE receipt_id=? AND name='A'`, old))
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id=?`, old)
	check(500, false)
	live := admitOccurrenceFixture(t, s, duet, 1700000200, "projection-live")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_received_artist_credits WHERE receipt_id=? AND name='A'`, live))
	plan := enrichmentQueryPlan(t, s, scopedArtistClaim, 1, 500)
	require.NotContains(t, plan, "ingest_events")
	require.Contains(t, plan, "idx_enrichment_received_credit (profile_id=? AND occurrence_id=? AND name=?)")
}

// TestEnrichmentArtistDeleteScope checks indexed cleanup amid unrelated claims
// and repeated listens, including the occurrence cascade with standalone credits.
func TestEnrichmentArtistDeleteScope(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `INSERT INTO recordings(id) VALUES(900);
		INSERT INTO occurrences(id,recording_id,presented_title) VALUES(900,900,'First'),(901,900,'Second');
		WITH RECURSIVE ids(n) AS (VALUES(900) UNION ALL SELECT n+1 FROM ids WHERE n<4900)
		INSERT INTO artists(id) SELECT n FROM ids;
		INSERT INTO occurrence_credits VALUES(900,0,'Artist',900);
		INSERT INTO occurrence_credits SELECT 901,id,printf('Artist %d',id),id FROM artists WHERE id>900;
		WITH RECURSIVE ids(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM ids WHERE n<2000)
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) SELECT 1,900,n,'scale-'||n FROM ids;
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,901,1,'unrelated');
		INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind)
		SELECT 1,id,CASE WHEN id=900 THEN 'scale-1' ELSE 'unrelated' END,'fixture','22222222-2222-4222-8222-222222222222',
			'navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM artists WHERE id>=900`)
	plan := enrichmentQueryPlan(t, s, `DELETE FROM listens WHERE event_id=?`, "scale-1")
	require.Contains(t, plan, "idx_metadata_event_fk (event_id=?)")
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND track_id=900`)
	require.Equal(t, 4000, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`))
}

// TestEnrichmentArtistBridgeDelete covers the track cascade that removes the
// artist_tracks rows before the bridged occurrence is deleted.
func TestEnrichmentArtistBridgeDelete(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) SELECT DISTINCT l.profile_id,a.artist_id,l.event_id,'test','22222222-2222-4222-8222-222222222222','navidrome-subsonic','2026-01-02T00:00:00Z','mbid'
		FROM artist_tracks a JOIN listens l ON l.track_id=a.track_id WHERE a.track_id=1`)
	require.Positive(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL`))
	execSQL(t, s, `DELETE FROM tracks WHERE id=1`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values c WHERE c.artist_id IS NOT NULL AND NOT EXISTS (
		SELECT 1 FROM artist_tracks a JOIN listens l ON l.track_id=a.track_id
		WHERE a.artist_id=c.artist_id AND l.profile_id=c.profile_id)`))
}

// enrichmentQueryPlan reports SQLite's access paths without wall-clock assertions.
func enrichmentQueryPlan(t *testing.T, s *Sqlite, query string, args ...any) string {
	t.Helper()
	rows, err := s.db.QueryContext(t.Context(), "EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var id, parent, unused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
		plan.WriteString(detail + "\n")
	}
	require.NoError(t, rows.Err())
	return plan.String()
}
