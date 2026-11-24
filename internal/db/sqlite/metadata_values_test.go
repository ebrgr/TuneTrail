package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestMetadataValueBindings validates the entity discriminator and the event's profile and credits.
func TestMetadataValueBindings(t *testing.T) {
	s := mergeFixture(t)
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE profile_id=1 AND track_id=1`).Scan(&event))
	insert := `INSERT INTO metadata_values(profile_id,artist_id,release_id,event_id,kind,value,instance,source,observed_at)
		VALUES(?,?,?,?,'genre','Rock','fixture','source-a','2026-01-02T00:00:00Z')`
	for _, tc := range []struct {
		args []any
		err  string
	}{
		{[]any{1, nil, nil, event}, "Metadata event does not support"},
		{[]any{1, 1, nil, nil}, "NOT NULL constraint failed"},
		{[]any{1, 1, 1, event}, "CHECK constraint failed"},
		{[]any{1, 999, nil, event}, "Metadata event does not support"},
		{[]any{1, nil, 999, event}, "Metadata event does not support"},
		{[]any{2, 1, nil, event}, "Metadata event does not support"},
		{[]any{1, 2, nil, event}, "Metadata event does not support"},
		{[]any{1, nil, 2, event}, "Metadata event does not support"},
	} {
		require.ErrorContains(t, s.Exec(insert, tc.args...), tc.err)
	}
	execSQL(t, s, insert, 1, 1, nil, event)
	execSQL(t, s, insert, 1, nil, 1, event)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,artist_id,event_id,kind,value,instance,source,observed_at)
		VALUES(1,1,?,'genre','Jazz','fixture','source-b','2026-01-02T00:00:00Z')`, event)
	var otherProfile string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE profile_id=2 AND track_id=1`).Scan(&otherProfile))
	execSQL(t, s, insert, 2, 1, nil, otherProfile)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,200,'same-artist')`)
	for _, replacement := range []any{nil, "same-artist", otherProfile} {
		require.ErrorContains(t, s.Exec(`UPDATE metadata_values SET event_id=? WHERE event_id=?`, replacement, event), "binding is immutable")
	}
	require.ErrorContains(t, s.Exec(`UPDATE metadata_values SET artist_id=2 WHERE event_id=? AND artist_id=1`, event), "binding is immutable")
	require.NoError(t, s.DeleteListenEvent(db.WithProfile(t.Context(), 1), event))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE profile_id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE event_id='same-artist'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE profile_id=2`))
	assertForeignKeys(t, s)
}

// TestMetadataCreditRemoval deletes only observations tied to the removed credit.
func TestMetadataCreditRemoval(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO artist_tracks VALUES(2,1,0);
		INSERT INTO metadata_values(profile_id,artist_id,event_id,kind,value,instance,source,observed_at)
		SELECT 1,2,event_id,'genre','Rock','fixture','test','2026-01-02T00:00:00Z'
		FROM listens WHERE profile_id=1 AND track_id IN (1,2)`)
	require.NoError(t, s.UpdateTrack(t.Context(), db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN listens l USING(event_id)
		WHERE m.artist_id=2 AND l.track_id=2`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN listens l USING(event_id)
		WHERE m.artist_id=2 AND l.track_id=1`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM listens`))
}

// TestMetadataRecordingMerge keeps each declaration on its own occurrence through merge and undo.
func TestMetadataRecordingMerge(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,kind,value,instance,source,observed_at)
		SELECT 1,o.recording_id,o.id,l.event_id,'mbid',printf('value-%d',o.id),'fixture','source-a','2026-01-02T00:00:00Z'
		FROM occurrences o JOIN listens l ON l.track_id=o.id WHERE l.profile_id=1`)
	check := func(want int) {
		t.Helper()
		require.Equal(t, want, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM metadata_values`))
		require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN occurrences o
			ON o.id=m.occurrence_id AND o.recording_id=m.recording_id JOIN listens l
			ON l.event_id=m.event_id AND l.track_id=o.id`))
	}
	check(2)
	require.NoError(t, s.MergeTracks(t.Context(), 1, 2))
	check(1)
	require.NoError(t, s.UndoTrackMerge(t.Context(), 1, 2))
	check(2)
	assertForeignKeys(t, s)
}

// TestMetadataReleaseMerge preserves the origin when its occurrence moves to the new release.
func TestMetadataReleaseMerge(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,release_id,event_id,kind,value,instance,source,observed_at)
		SELECT 1,1,event_id,'release_date','2006','fixture','source-a','2026-01-02T00:00:00Z'
		FROM listens WHERE profile_id=1 AND track_id=1`)
	require.NoError(t, s.MergeAlbums(t.Context(), 1, 2, false))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN listens l USING(event_id)
		JOIN occurrences o ON o.id=l.track_id AND o.release_id=m.release_id
		WHERE m.release_id=2 AND m.value='2006'`))
	assertForeignKeys(t, s)
}

// TestMetadataLeaseRecovery reopens the database after a worker leaves an expired lease.
func TestMetadataLeaseRecovery(t *testing.T) {
	s, _, path := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, s, 1, "metadata-recovery", "Recovery", 1)
	require.NoError(t, s.ConfigureEnrichment(t.Context(), true))
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Recovery", Artist: "Artist", Release: "Album"}, 1700000000, "recover-metadata")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	claim, err := s.claimEnrichment(t.Context())
	require.NoError(t, err)
	require.Equal(t, id, claim.receipt)
	require.NotEmpty(t, claim.lease)
	execSQL(t, s, `UPDATE enrichment_jobs SET lease_until=0 WHERE receipt_id=?`, id)
	require.NoError(t, s.db.Close())

	conn, err := sql.Open("sqlite", sqliteURI(path)+"?_pragma=foreign_keys(ON)")
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	reopened := &Sqlite{db: conn}
	found, err := reopened.ResolveNextEnrichment(t.Context(), enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
			{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Now()},
		}}
	}})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, scalar(t, reopened, `SELECT count(*) FROM metadata_values m JOIN ingest_events e USING(profile_id,event_id)
		JOIN enrichment_jobs j USING(receipt_id)
		WHERE e.receipt_id=? AND j.attempts=2 AND j.state='found' AND e.state='accepted'`, id))
}
