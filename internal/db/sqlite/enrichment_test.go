package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

type enrichmentFixture struct {
	lookup func(context.Context, enrichment.Request) enrichment.Result
}

// configuredEnrichmentFixture enables queue admission for these isolated tests.
func configuredEnrichmentFixture(t *testing.T) *Sqlite {
	t.Helper()
	s := convertedOccurrenceFixture(t)
	require.NoError(t, s.ConfigureEnrichment(t.Context(), true))
	return s
}

// TestEnrichmentUpgrade leaves existing history untouched and does not enqueue a backfill.
func TestEnrichmentUpgrade(t *testing.T) {
	s, provider, _ := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE mode='shadow' AND event_id IS NULL`)
	_, err := provider.UpTo(t.Context(), 23)
	require.NoError(t, err)
	queries := []string{
		`SELECT * FROM listens ORDER BY event_id`,
		`SELECT receipt_id,original_snapshot FROM ingest_events ORDER BY receipt_id`,
		`SELECT id,musicbrainz_id FROM artists ORDER BY id`,
		`SELECT id,musicbrainz_id,mbid_source FROM recordings ORDER BY id`,
	}
	before := make([][32]byte, len(queries))
	for i, query := range queries {
		before[i] = queryDigest(t, s.db, query)
	}
	_, err = provider.UpTo(t.Context(), currentSchemaVersion)
	require.NoError(t, err)
	for i, query := range queries {
		require.Equal(t, before[i], queryDigest(t, s.db, query))
	}
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_jobs`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values`))
	require.NoError(t, checkDatabase(s.db))
}

func (enrichmentFixture) Instance() string { return "test-instance" }
func (f enrichmentFixture) Lookup(ctx context.Context, r enrichment.Request) enrichment.Result {
	return f.lookup(ctx, r)
}

// TestEnrichmentPersistsScopedDeclarations confirms the receipt is immutable and external IDs stay scoped.
func TestEnrichmentPersistsScopedDeclarations(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Enrichment title", Artist: "Enrichment artist", Release: "Enrichment album"}, 1700000000, "enrich")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	requireReceiptState(t, s, id, "accepted")
	var before string
	require.NoError(t, s.db.QueryRow(`SELECT original_snapshot FROM ingest_events WHERE receipt_id=?`, id).Scan(&before))
	provider := enrichmentFixture{lookup: func(_ context.Context, r enrichment.Request) enrichment.Result {
		require.EqualValues(t, 1, r.ProfileID)
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
			{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Now()},
			{Entity: "artist", Name: "Enrichment artist", Field: "musicbrainz_id", Value: "e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59", Source: "navidrome-subsonic", ObservedAt: time.Now()},
		}}
	}}
	found, err := s.ResolveNextEnrichment(t.Context(), provider)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN ingest_events e
		ON e.profile_id=m.profile_id AND e.event_id=m.event_id WHERE e.receipt_id=?`, id))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND instance='test-instance' AND value=?`, fixtureRecording))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND instance='test-instance'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id<>1`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM recordings WHERE musicbrainz_id=?`, fixtureRecording))
	var after, state string
	require.NoError(t, s.db.QueryRow(`SELECT e.original_snapshot,j.state FROM ingest_events e JOIN enrichment_jobs j USING(receipt_id) WHERE e.receipt_id=?`, id).Scan(&after, &state))
	require.Equal(t, before, after)
	require.Equal(t, "found", state)
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var exported bool
	for _, item := range page {
		if item.TrackMbid != nil && item.TrackMbid.String() == fixtureRecording {
			exported = true
			require.Equal(t, "navidrome-subsonic", item.MBIDSource)
			require.NotEmpty(t, item.MBIDDeclaredAt)
		}
	}
	require.True(t, exported)
	var occurrence int32
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id FROM ingest_events WHERE receipt_id=?`, id).Scan(&occurrence))
	track, err := s.GetTrack(db.WithProfile(t.Context(), 1), db.GetTrackOpts{ID: occurrence})
	require.NoError(t, err)
	require.NotNil(t, track.MbzID)
	require.Equal(t, fixtureRecording, track.MbzID.String())
	var artistID int32
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`).Scan(&artistID))
	artist, err := s.GetArtist(db.WithProfile(t.Context(), 1), db.GetArtistOpts{ID: artistID})
	require.NoError(t, err)
	require.NotNil(t, artist.MbzID)
	require.Equal(t, "e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59", artist.MbzID.String())
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(2,?,1700000040,'legacy')`, occurrence)
	track, err = s.GetTrack(db.WithProfile(t.Context(), 2), db.GetTrackOpts{ID: occurrence})
	require.NoError(t, err)
	require.Nil(t, track.MbzID)
	artist, err = s.GetArtist(db.WithProfile(t.Context(), 2), db.GetArtistOpts{ID: artistID})
	require.NoError(t, err)
	require.Nil(t, artist.MbzID)
	track, err = s.GetTrack(db.WithoutProfile(t.Context()), db.GetTrackOpts{ID: occurrence, AllowUnplayed: true})
	require.NoError(t, err)
	require.Nil(t, track.MbzID)
	found, err = s.ResolveNextEnrichment(t.Context(), provider)
	require.NoError(t, err)
	require.False(t, found)
}

// TestEnrichmentDeletionDuringLookup ensures a stale network result cannot recreate deleted data.
func TestEnrichmentDeletionDuringLookup(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	s.db.SetMaxOpenConns(2)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Delete title", Artist: "Delete artist", Release: "Delete album"}, 1700000010, "delete-enrich")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	started, finish := make(chan struct{}), make(chan struct{})
	provider := enrichmentFixture{lookup: func(ctx context.Context, _ enrichment.Request) enrichment.Result {
		close(started)
		<-finish
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Now()}}}
	}}
	done := make(chan error, 1)
	go func() { _, err := s.ResolveNextEnrichment(context.Background(), provider); done <- err }()
	<-started
	execSQL(t, s, `DELETE FROM listens WHERE event_id=(SELECT event_id FROM ingest_events WHERE receipt_id=?)`, id)
	close(finish)
	require.NoError(t, <-done)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL`))
}

// TestEnrichmentArtistReassignmentDuringLookup rejects a replacement identity with the same name.
func TestEnrichmentArtistReassignmentDuringLookup(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	s.db.SetMaxOpenConns(2)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Race title", Artist: "Race artist", Release: "Race album"}, 1700000060, "artist-race")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	var occurrence, oldArtist, newArtist int64
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id FROM ingest_events WHERE receipt_id=?`, id).Scan(&occurrence))
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM artist_aliases WHERE alias='Race artist'`).Scan(&oldArtist))
	require.NoError(t, s.db.QueryRow(`INSERT INTO artists DEFAULT VALUES RETURNING id`).Scan(&newArtist))
	started, finish := make(chan struct{}), make(chan struct{})
	provider := enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
		close(started)
		<-finish
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "artist", Name: "Race artist", Field: "musicbrainz_id", Value: "e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59", Source: "navidrome-subsonic", ObservedAt: time.Now()}}}
	}}
	done := make(chan error, 1)
	go func() { _, err := s.ResolveNextEnrichment(context.Background(), provider); done <- err }()
	<-started
	execSQL(t, s, `DELETE FROM artists WHERE id=?`, oldArtist)
	execSQL(t, s, `INSERT INTO artist_aliases(artist_id,alias,source,is_primary) VALUES(?,'Race artist','Canonical',1)`, newArtist)
	execSQL(t, s, `UPDATE occurrence_credits SET artist_id=? WHERE occurrence_id=? AND name='Race artist'`, newArtist, occurrence)
	close(finish)
	require.NoError(t, <-done)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL`))
}

// TestEnrichmentCatalogEditDuringLookup rejects a result based on stale metadata.
func TestEnrichmentCatalogEditDuringLookup(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	s.db.SetMaxOpenConns(2)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Edit title", Artist: "Edit artist", Release: "Edit album"}, 1700000070, "edit-race")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	started, finish := make(chan struct{}), make(chan struct{})
	provider := enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
		close(started)
		<-finish
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Now()}}}
	}}
	done := make(chan error, 1)
	go func() { _, err := s.ResolveNextEnrichment(context.Background(), provider); done <- err }()
	<-started
	execSQL(t, s, `UPDATE occurrences SET presented_title='Edited' WHERE id=(SELECT occurrence_id FROM ingest_events WHERE receipt_id=?)`, id)
	close(finish)
	require.NoError(t, <-done)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state='failed'`, id))
}

// TestEnrichmentDisabledAndExpiredLease verifies finite recovery without ingest retries.
func TestEnrichmentDisabledAndExpiredLease(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Disabled title", Artist: "Disabled artist", Release: "Disabled album"}, 1700000020, "disabled-enrich")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	found, err := s.ResolveNextEnrichment(t.Context(), nil)
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state='disabled'`, id))
	execSQL(t, s, `UPDATE enrichment_jobs SET state='working',lease='expired',lease_until=0,attempts=2 WHERE receipt_id=?`, id)
	found, err = s.ResolveNextEnrichment(t.Context(), nil)
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state='failed' AND lease IS NULL`, id))
	require.NoError(t, s.ConfigureEnrichment(t.Context(), false))
	id2 := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "No queue", Artist: "Disabled artist", Release: "Disabled album"}, 1700000021, "disabled-gate")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=?`, id2))
}

// TestEnrichmentPreservesReceivedIDs rejects claims for entities already identified by the event.
func TestEnrichmentPreservesReceivedIDs(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	snapshot := ingest.Snapshot{Title: "Known title", Artist: "Known artist", Release: "Known album", Recording: fixtureRecording, ArtistMBIDs: []string{"e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59"}}
	id := admitOccurrenceFixture(t, s, snapshot, 1700000030, "known-enrich")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	provider := enrichmentFixture{lookup: func(_ context.Context, _ enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
			{Entity: "recording", Field: "musicbrainz_id", Value: "d10c78f5-5ad3-4ba2-bd43-f4ac12c62f59", Source: "navidrome-subsonic", ObservedAt: time.Now()},
			{Entity: "artist", Name: "Known artist", Field: "musicbrainz_id", Value: "c10c78f5-5ad3-4ba2-bd43-f4ac12c62f59", Source: "navidrome-subsonic", ObservedAt: time.Now()},
		}}
	}}
	found, err := s.ResolveNextEnrichment(t.Context(), provider)
	require.True(t, found)
	require.NoError(t, err)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state='found'`, id))
}

// TestEnrichmentArtistMergePreservesConflictingClaims keeps both scoped assertions for review.
func TestEnrichmentArtistMergePreservesConflictingClaims(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	receipt := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Merge title", Artist: "Merge artist", Release: "Merge album"}, 1700000050, "merge-enrich")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	var from, to int32
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM artist_aliases WHERE alias='Merge artist'`).Scan(&from))
	require.NoError(t, s.db.QueryRow(`INSERT INTO artists DEFAULT VALUES RETURNING id`).Scan(&to))
	execSQL(t, s, `INSERT INTO artist_tracks(artist_id,track_id,is_primary)
		SELECT ?,occurrence_id,0 FROM ingest_events WHERE receipt_id=?`, to, receipt)
	_, err := s.db.Exec(`INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES
		(1,?, (SELECT event_id FROM ingest_events WHERE receipt_id=?), 'instance','e10c78f5-5ad3-4ba2-bd43-f4ac12c62f59','navidrome-subsonic','2026-01-01T00:00:00Z','mbid'),
		(1,?, (SELECT event_id FROM ingest_events WHERE receipt_id=?), 'instance','c10c78f5-5ad3-4ba2-bd43-f4ac12c62f59','navidrome-subsonic','2026-01-01T00:00:00Z','mbid')`, from, receipt, to, receipt)
	require.NoError(t, err)
	require.NoError(t, s.MergeArtists(t.Context(), from, to, false))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND artist_id=? AND profile_id=1`, to))
}
