package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

const knownArtist174 = "11111111-1111-4111-8111-111111111111"
const enrichedArtist174 = "22222222-2222-4222-8222-222222222222"

// TestEnrichmentArchiveForeignCatalog restores claims without promoting them to global identity.
func TestEnrichmentArchiveForeignCatalog(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Archive song", Artist: "Archive artist", Release: "Archive album"}, 1700000400, "archive-claim")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	provider := enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
			{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
			{Entity: "artist", Name: "Archive artist", Field: "musicbrainz_id", Value: enrichedArtist174, Source: "navidrome-subsonic", ObservedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		}}
	}}
	_, err := s.ResolveNextEnrichment(t.Context(), provider)
	require.NoError(t, err)
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, id).Scan(&event))
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var item *db.ExportItem
	for _, candidate := range page {
		if candidate.Cursor == event {
			item = candidate
		}
	}
	require.NotNil(t, item)
	for _, owner := range []bool{false, true} {
		t.Run(map[bool]string{false: "root", true: "owner"}[owner], func(t *testing.T) {
			target, _, _ := migrationDB(t, currentSchemaVersion)
			testfixtures.Profile(t, target, 1, "restore", "Restore", 1)
			ctx := t.Context()
			if owner {
				ctx = db.WithImportedCatalog(ctx, 1)
			}
			count, err := target.RestoreArchive(ctx, 1, []*db.ExportItem{item})
			require.NoError(t, err)
			require.Equal(t, 1, count)
			require.Zero(t, scalar(t, target, `SELECT count(*) FROM artists WHERE musicbrainz_id=?`, enrichedArtist174))
			require.Zero(t, scalar(t, target, `SELECT count(*) FROM recordings WHERE musicbrainz_id=?`, fixtureRecording))
			require.Equal(t, 1, scalar(t, target, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND value=? AND source='navidrome-subsonic'`, enrichedArtist174))
			require.Equal(t, 1, scalar(t, target, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND value=? AND source='navidrome-subsonic'`, fixtureRecording))
			got, err := target.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 10})
			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Equal(t, "navidrome-subsonic", got[0].Artists[0].MBIDSource)
			require.Equal(t, "2026-01-02T00:00:00Z", got[0].Artists[0].MBIDDeclaredAt)
		})
	}
}

// TestEnrichmentConflictingArtistClaimsArchive retains conflicts without selecting either ID.
func TestEnrichmentConflictingArtistClaimsArchive(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	receipt := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Conflict song", Artist: "Conflict artist", Release: "Conflict album"}, 1700000800, "conflicting-claims")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	var artist int64
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM artist_aliases WHERE alias='Conflict artist'`).Scan(&artist))
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES
		(1,?,(SELECT event_id FROM ingest_events WHERE receipt_id=?),'server-a',?,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid'),
		(1,?,(SELECT event_id FROM ingest_events WHERE receipt_id=?),'server-b',?,'navidrome-subsonic','2026-01-03T00:00:00Z','mbid'),
		(1,?,(SELECT event_id FROM ingest_events WHERE receipt_id=?),'server-c',?,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid')`, artist, receipt, knownArtist174, artist, receipt, enrichedArtist174, artist, receipt, knownArtist174)
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&event))
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var item *db.ExportItem
	for _, candidate := range page {
		if candidate.Cursor == event {
			item = candidate
		}
	}
	require.NotNil(t, item)
	require.Len(t, item.Artists, 1)
	require.Nil(t, item.Artists[0].MbzID)
	require.Len(t, item.Artists[0].MBIDDeclarations, 3)
	target, _, _ := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, target, 1, "conflict-restore", "Restore", 1)
	count, err := target.RestoreArchive(t.Context(), 1, []*db.ExportItem{item})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, 3, scalar(t, target, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`))
	got, err := target.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 10})
	require.NoError(t, err)
	require.Nil(t, got[0].Artists[0].MbzID)
	require.Len(t, got[0].Artists[0].MBIDDeclarations, 3)
}

// TestEnrichmentIndependentCredits preserves A's incoming ID and makes B's claim visible.
func TestEnrichmentIndependentCredits(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	snapshot := ingest.Snapshot{Title: "Duet", Artist: "A & B", Release: "Duets", ArtistNames: []string{"A", "B"}, ArtistMbidMappings: []ingest.ArtistMapping{{Name: "A", MBID: knownArtist174}}}
	receipt := admitOccurrenceFixture(t, s, snapshot, 1700000100, "duet")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	provider := enrichmentFixture{lookup: func(_ context.Context, r enrichment.Request) enrichment.Result {
		require.Equal(t, []string{"B"}, enrichment.MissingArtistNames(r.Snapshot))
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{{Entity: "artist", Name: "B", Field: "musicbrainz_id", Value: enrichedArtist174, Source: "navidrome-subsonic", ObservedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}}}
	}}
	found, err := s.ResolveNextEnrichment(t.Context(), provider)
	require.True(t, found)
	require.NoError(t, err)
	var b int32
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM artist_aliases WHERE alias='B'`).Scan(&b))
	artist, err := s.GetArtist(db.WithProfile(t.Context(), 1), db.GetArtistOpts{ID: b})
	require.NoError(t, err)
	require.NotNil(t, artist.MbzID)
	require.Equal(t, enrichedArtist174, artist.MbzID.String())
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&event))
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var matched bool
	var exported *db.ExportItem
	for _, item := range page {
		if item.Cursor != event {
			continue
		}
		matched = true
		exported = item
		for _, credit := range item.Artists {
			switch credit.Name {
			case "A":
				require.NotNil(t, credit.MbzID)
				require.Equal(t, knownArtist174, credit.MbzID.String())
				require.Equal(t, "source-event", credit.MBIDSource)
			case "B":
				require.NotNil(t, credit.MbzID)
				require.Equal(t, enrichedArtist174, credit.MbzID.String())
				require.Equal(t, "navidrome-subsonic", credit.MBIDSource)
				require.Equal(t, "2026-01-02T00:00:00Z", credit.MBIDDeclaredAt)
			}
		}
	}
	require.True(t, matched)
	target, _, _ := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, target, 1, "duet-restore", "Restore", 1)
	count, err := target.RestoreArchive(t.Context(), 1, []*db.ExportItem{exported})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Zero(t, scalar(t, target, `SELECT count(*) FROM artists WHERE musicbrainz_id IN (?,?)`, knownArtist174, enrichedArtist174))
	require.Equal(t, 1, scalar(t, target, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND value=? AND source='source-event'`, knownArtist174))
	require.Equal(t, 1, scalar(t, target, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND value=? AND source='navidrome-subsonic'`, enrichedArtist174))
	// A later shared catalog ID must not erase a different ID received in this event.
	execSQL(t, s, `UPDATE artists SET musicbrainz_id='33333333-3333-4333-8333-333333333333' WHERE id=(SELECT artist_id FROM artist_aliases WHERE alias='A')`)
	page, err = s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	for _, item := range page {
		if item.Cursor != event {
			continue
		}
		for _, credit := range item.Artists {
			if credit.Name == "A" {
				require.Equal(t, "33333333-3333-4333-8333-333333333333", credit.MbzID.String())
				require.Len(t, credit.MBIDDeclarations, 1)
				require.Equal(t, knownArtist174, credit.MBIDDeclarations[0].MBID.String())
				require.Equal(t, "source-event", credit.MBIDDeclarations[0].Source)
			}
		}
	}
}

// TestEnrichmentExportPreservesReceivedRecording keeps an event ID alongside a different catalog ID.
func TestEnrichmentExportPreservesReceivedRecording(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	receipt := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Recording conflict", Artist: "Artist", Release: "Album", Recording: fixtureRecording}, 1700000900, "recording-event")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	execSQL(t, s, `UPDATE recordings SET musicbrainz_id='33333333-3333-4333-8333-333333333333' WHERE id=(SELECT recording_id FROM ingest_events WHERE receipt_id=?)`, receipt)
	var event string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&event))
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var matched bool
	for _, item := range page {
		if item.Cursor == event {
			matched = true
			require.Equal(t, "33333333-3333-4333-8333-333333333333", item.TrackMbid.String())
			require.Len(t, item.RecordingDeclarations, 1)
			require.Equal(t, fixtureRecording, item.RecordingDeclarations[0].MBID.String())
			require.Equal(t, "source-event", item.RecordingDeclarations[0].Source)
		}
	}
	require.True(t, matched)
}

// TestEnrichmentRetention keeps values after receipt removal and deletes them with their listen.
func TestEnrichmentRetention(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	snapshot := ingest.Snapshot{Title: "Retention", Artist: "Retained artist", Release: "Retained album"}
	first := admitOccurrenceFixture(t, s, snapshot, 1700000100, "retention-one")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	provider := enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
		return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
			{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: time.Now()},
			{Entity: "artist", Name: "Retained artist", Field: "musicbrainz_id", Value: enrichedArtist174, Source: "navidrome-subsonic", ObservedAt: time.Now()},
		}}
	}}
	_, err := s.ResolveNextEnrichment(t.Context(), provider)
	require.NoError(t, err)
	second := admitOccurrenceFixture(t, s, snapshot, 1700000200, "retention-two")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	var occurrence int64
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id FROM ingest_events WHERE receipt_id=?`, first).Scan(&occurrence))
	var originalEvent, laterEvent string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, first).Scan(&originalEvent))
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM ingest_events WHERE receipt_id=?`, second).Scan(&laterEvent))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND track_id=?`, occurrence))
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id=?`, first)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND occurrence_id=?`, occurrence))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`))
	page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var checked int
	for _, item := range page {
		switch item.Cursor {
		case originalEvent:
			require.Len(t, item.RecordingDeclarations, 1)
			require.Len(t, item.Artists[0].MBIDDeclarations, 1)
			checked++
		case laterEvent:
			require.Empty(t, item.RecordingDeclarations)
			require.Empty(t, item.Artists[0].MBIDDeclarations)
			checked++
		}
	}
	require.Equal(t, 2, checked)
	var artist int64
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`).Scan(&artist))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin,event_id) VALUES(2,?,1700000101,'legacy','other-retention')`, occurrence)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES(2,?,'other-retention','other-instance',?,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid')`, artist, enrichedArtist174)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,instance,value,source,observed_at,kind) SELECT 2,recording_id,id,'other-retention','other-instance',?2,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM occurrences WHERE id=?1`, occurrence, fixtureRecording)
	require.NoError(t, s.DeleteListenEvent(db.WithProfile(t.Context(), 1), originalEvent))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE event_id=?`, laterEvent))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND occurrence_id=?`, occurrence))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=2`))
}

// TestEnrichmentRetentionAcrossOccurrences removes a claim with its originating listen.
func TestEnrichmentRetentionAcrossOccurrences(t *testing.T) {
	s := configuredEnrichmentFixture(t)
	receipt := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Edition one", Artist: "Artist", Release: "Album"}, 1700000500, "edition-one")
	resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
	var occurrence, recording, other int64
	require.NoError(t, s.db.QueryRow(`SELECT occurrence_id,recording_id FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&occurrence, &recording))
	require.NoError(t, s.db.QueryRow(`INSERT INTO occurrences(recording_id,presented_title) VALUES(?,'Edition two') RETURNING id`, recording).Scan(&other))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,?,1700000501,'legacy')`, other)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,instance,value,source,observed_at,kind) SELECT 1,recording_id,id,(SELECT event_id FROM listens WHERE track_id=?1 AND profile_id=1),'server',?2,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM occurrences WHERE id=?1`, occurrence, fixtureRecording)
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND track_id=?`, occurrence)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND value=?`, fixtureRecording))
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND track_id=?`, other)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND value=?`, fixtureRecording))
}

// TestEnrichmentOccurrenceDeleteRemovesOrigin keeps the second edition without borrowing its listen.
func TestEnrichmentOccurrenceDeleteRemovesOrigin(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var recording, first, second int64
	require.NoError(t, s.db.QueryRow(`INSERT INTO recordings DEFAULT VALUES RETURNING id`).Scan(&recording))
	require.NoError(t, s.db.QueryRow(`INSERT INTO occurrences(recording_id,presented_title) VALUES(?,'First') RETURNING id`, recording).Scan(&first))
	require.NoError(t, s.db.QueryRow(`INSERT INTO occurrences(recording_id,presented_title) VALUES(?,'Second') RETURNING id`, recording).Scan(&second))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(1,?,1700000600,'legacy'),(1,?,1700000601,'legacy')`, first, second)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,instance,value,source,observed_at,kind) SELECT 1,recording_id,id,(SELECT event_id FROM listens WHERE track_id=?1 AND profile_id=1),'server',?2,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM occurrences WHERE id=?1`, first, fixtureRecording)
	execSQL(t, s, `DELETE FROM occurrences WHERE id=?`, first)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND value=?`, fixtureRecording))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE track_id=?`, second))
}

// TestEnrichmentProfileDeletionRemovesClaims verifies privacy during cascading deletion.
func TestEnrichmentProfileDeletionRemovesClaims(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	var recording, first, second int64
	require.NoError(t, s.db.QueryRow(`INSERT INTO recordings DEFAULT VALUES RETURNING id`).Scan(&recording))
	require.NoError(t, s.db.QueryRow(`INSERT INTO occurrences(recording_id,presented_title) VALUES(?,'First') RETURNING id`, recording).Scan(&first))
	require.NoError(t, s.db.QueryRow(`INSERT INTO occurrences(recording_id,presented_title) VALUES(?,'Second') RETURNING id`, recording).Scan(&second))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(2,?,1700000700,'legacy'),(2,?,1700000701,'legacy')`, first, second)
	execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,instance,value,source,observed_at,kind) SELECT 2,recording_id,id,(SELECT event_id FROM listens WHERE track_id=?1 AND profile_id=2),'server',?2,'navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM occurrences WHERE id=?1`, first, fixtureRecording)
	execSQL(t, s, `DELETE FROM listening_profiles WHERE id=2`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=2`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=2`))
}
