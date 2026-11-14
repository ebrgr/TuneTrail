package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestTrackAlbumParticipation tests ranking, identity matching and automatic removal in unchanged v3.
func TestTrackAlbumParticipation(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{2}}))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE artist_id=2 AND release_id=1`))
	album, err := s.GetAlbum(ctx, db.GetAlbumOpts{ArtistID: 2, Title: "One"})
	require.NoError(t, err)
	require.EqualValues(t, 1, album.ID)
	artists, err := s.GetArtistsForAlbum(ctx, 1)
	require.NoError(t, err)
	require.Len(t, artists, 2)
	for _, profile := range []int32{1, 2} {
		opts := db.GetItemsOpts{ArtistID: 2, Page: 1, Limit: 10, Timeframe: db.Timeframe{From: time.Unix(1, 0), To: time.Unix(200, 0)}}
		page, err := s.GetTopAlbumsPaginated(db.WithProfile(ctx, profile), opts)
		require.NoError(t, err)
		require.Len(t, page.Items, 2)
	}
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}))
	for _, profile := range []int32{1, 2} {
		page, err := s.GetTopAlbumsPaginated(db.WithProfile(ctx, profile), db.GetItemsOpts{ArtistID: 2, Page: 1, Limit: 10, Timeframe: db.Timeframe{From: time.Unix(1, 0), To: time.Unix(200, 0)}})
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.EqualValues(t, 2, page.Items[0].Item.ID)
	}
	require.Equal(t, 4, scalar(t, s, `SELECT COUNT(*) FROM listens`))
	require.NoError(t, checkIntegrity(s.db))
	require.Equal(t, 3, scalar(t, s, `SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1`))
}

// TestAlbumCreditsSurviveTrackEdits covers legacy, both credit orders, primary promotion and partial albums.
func TestAlbumCreditsSurviveTrackEdits(t *testing.T) {
	for _, mode := range []string{"explicit_first", "explicit_last", "legacy", "primary", "other_track"} {
		t.Run(mode, func(t *testing.T) {
			s := mergeFixture(t)
			ctx := context.Background()
			if mode == "explicit_first" {
				require.NoError(t, s.AddArtistsToAlbum(ctx, db.AddArtistsToAlbumOpts{AlbumID: 1, ArtistIDs: []int32{2}}))
			}
			if mode == "legacy" {
				require.NoError(t, s.Exec(`INSERT INTO artist_releases VALUES(2,1,0)`))
			}
			require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{2}}))
			switch mode {
			case "explicit_last":
				require.NoError(t, s.AddArtistsToAlbum(ctx, db.AddArtistsToAlbumOpts{AlbumID: 1, ArtistIDs: []int32{2}}))
			case "primary":
				require.NoError(t, s.SetPrimaryAlbumArtist(ctx, 1, 2, true))
				require.NoError(t, s.SetPrimaryAlbumArtist(ctx, 1, 1, true))
			case "other_track":
				require.NoError(t, s.Exec(`INSERT INTO tracks(id,release_id) VALUES(50,1)`))
				require.NoError(t, s.AddArtistsToTrack(ctx, db.AddArtistsToTrackOpts{TrackID: 50, ArtistIDs: []int32{2}}))
			}
			album, err := s.GetAlbum(ctx, db.GetAlbumOpts{ID: 1})
			require.NoError(t, err)
			require.Len(t, album.Artists, 2)
			// Independent plus derived origins must count every listen just once.
			page, err := s.GetTopAlbumsPaginated(db.WithProfile(ctx, 1), db.GetItemsOpts{ArtistID: 2, Page: 1, Limit: 10, Timeframe: db.Timeframe{From: time.Unix(1, 0), To: time.Unix(200, 0)}})
			require.NoError(t, err)
			for _, item := range page.Items {
				require.EqualValues(t, 1, item.Item.ListenCount)
			}
			require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}))
			album, err = s.GetAlbum(ctx, db.GetAlbumOpts{ID: 1})
			require.NoError(t, err)
			require.Len(t, album.Artists, 2)
			if mode == "other_track" {
				require.NoError(t, s.DeleteTrack(ctx, 50))
				album, err = s.GetAlbum(ctx, db.GetAlbumOpts{ID: 1})
				require.NoError(t, err)
				require.Len(t, album.Artists, 1)
			}
			require.NoError(t, checkIntegrity(s.db))
		})
	}
}

// TestParticipationDoesNotDeleteLastAlbumOrHistory exercises an album whose only credit comes from a track.
func TestParticipationDoesNotDeleteLastAlbumOrHistory(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.Exec(`INSERT INTO releases(id) VALUES(3); INSERT INTO release_aliases VALUES(3,'Partial','User',1); INSERT INTO tracks(id,release_id) VALUES(3,3); INSERT INTO track_aliases VALUES(3,'Partial','User',1); INSERT INTO artist_tracks VALUES(1,3,1); INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,3,100),(2,3,100)`))
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 3, AddArtists: []int32{2}}))
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 3, RemoveArtists: []int32{2}}))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM releases WHERE id=3`))
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE track_id=3`))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE release_id=3`))
	require.NoError(t, checkIntegrity(s.db))
}

// TestIngestedCreditsAndAlbumMergeFollowTracks prevents ingestion and merges from freezing participation.
func TestIngestedCreditsAndAlbumMergeFollowTracks(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		require.NoError(t, s.AddArtistsToAlbum(ctx, db.AddArtistsToAlbumOpts{Derived: true, AlbumID: 1, ArtistIDs: []int32{2}}))
	}
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{2}}))
	require.NoError(t, s.Exec(`INSERT INTO releases(id) VALUES(3); INSERT INTO release_aliases VALUES(3,'Destination','User',1)`))
	require.NoError(t, s.MergeAlbums(ctx, 1, 3, false))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE artist_id=2 AND release_id=3`))
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}))
	album, err := s.GetAlbum(ctx, db.GetAlbumOpts{ID: 3})
	require.NoError(t, err)
	require.Len(t, album.Artists, 1)
	require.NoError(t, checkIntegrity(s.db))
}

// TestParticipationRollback rejects a mixed edit without leaking either metadata or relationships.
func TestParticipationRollback(t *testing.T) {
	s := mergeFixture(t)
	before := mergeSnapshot(t, s)
	require.ErrorIs(t, s.UpdateTrack(context.Background(), db.UpdateTrackOpts{ID: 1, Duration: 999, AddArtists: []int32{2}, RemoveArtists: []int32{1}}), db.ErrConflict)
	require.Equal(t, before, mergeSnapshot(t, s))
}

// TestSavedTrackDoesNotCreateIndependentAlbumCredit covers the ingestion creation path.
func TestSavedTrackDoesNotCreateIndependentAlbumCredit(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	track, err := s.SaveTrack(ctx, db.SaveTrackOpts{AlbumID: 1, ArtistIDs: []int32{2}, Title: "New participation"})
	require.NoError(t, err)
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE artist_id=2 AND release_id=1`))
	require.NoError(t, s.DeleteTrack(ctx, track.ID))
	album, err := s.GetAlbum(ctx, db.GetAlbumOpts{ID: 1})
	require.NoError(t, err)
	require.Len(t, album.Artists, 1)
}

// TestReviewedLegacyRepairGuards exercises the documented single-pair offline repair on fictional data.
func TestReviewedLegacyRepairGuards(t *testing.T) {
	for _, mode := range []string{"residual", "primary", "supported", "last_credit"} {
		t.Run(mode, func(t *testing.T) {
			s := mergeFixture(t)
			require.NoError(t, s.Exec(`INSERT INTO artist_releases VALUES(2,1,0)`))
			switch mode {
			case "primary":
				require.NoError(t, s.Exec(`UPDATE artist_releases SET is_primary=1 WHERE artist_id=2 AND release_id=1`))
			case "supported":
				require.NoError(t, s.Exec(`INSERT INTO artist_tracks VALUES(2,1,0)`))
			case "last_credit":
				require.NoError(t, s.Exec(`DELETE FROM artist_releases WHERE artist_id=1 AND release_id=1`))
			}
			result, err := s.db.Exec(`DELETE FROM artist_releases WHERE artist_id=? AND release_id=? AND is_primary=0
   AND NOT EXISTS(SELECT 1 FROM artist_tracks a JOIN tracks t ON t.id=a.track_id WHERE a.artist_id=? AND t.release_id=?)
   AND EXISTS(SELECT 1 FROM artist_releases other WHERE other.release_id=? AND other.artist_id<>?)`, 2, 1, 2, 1, 1, 2)
			require.NoError(t, err)
			n, err := result.RowsAffected()
			require.NoError(t, err)
			if mode == "residual" {
				require.EqualValues(t, 1, n)
			} else {
				require.Zero(t, n)
			}
			require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM releases`))
			require.Equal(t, 4, scalar(t, s, `SELECT COUNT(*) FROM listens`))
			require.NoError(t, checkIntegrity(s.db))
		})
	}
}
