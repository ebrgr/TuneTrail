package sqlite

import (
	"context"
	"sync"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestTrackAliasIdentity checks exact aliases, inclusion of artists, and explicit ambiguity.
func TestTrackAliasIdentity(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.SaveTrackAliases(ctx, 1, []string{"Secondary"}, "User"))
	require.NoError(t, s.AddArtistsToTrack(ctx, db.AddArtistsToTrackOpts{TrackID: 1, ArtistIDs: []int32{2}}))
	for _, artists := range [][]int32{{1}, {2}, {1, 2}, {2, 1}, {2, 1, 2}} {
		track, err := s.GetTrack(ctx, db.GetTrackOpts{Title: "Secondary", ReleaseID: 1, ArtistIDs: artists})
		require.NoError(t, err)
		require.EqualValues(t, 1, track.ID)
		require.Equal(t, "One", track.Title)
	}
	_, err := s.GetTrack(ctx, db.GetTrackOpts{Title: "secondary", ReleaseID: 1, ArtistIDs: []int32{1}})
	require.ErrorIs(t, err, db.ErrNotFound)
	_, err = s.GetTrack(ctx, db.GetTrackOpts{Title: "Secondary", ReleaseID: 2, ArtistIDs: []int32{1}})
	require.ErrorIs(t, err, db.ErrNotFound)
	require.NoError(t, s.Exec(`INSERT INTO tracks(id,release_id,duration) VALUES(3,1,180)`))
	require.NoError(t, s.Exec(`INSERT INTO track_aliases VALUES(3,'Secondary','User',1)`))
	require.NoError(t, s.Exec(`INSERT INTO artist_tracks VALUES(1,3,1)`))
	_, err = s.GetTrack(ctx, db.GetTrackOpts{Title: "Secondary", ReleaseID: 1, ArtistIDs: []int32{1}})
	require.ErrorIs(t, err, db.ErrAmbiguousTrack)
}

// TestFillTrackIdentityConcurrent ensures competing fills cannot overwrite the winning MBID.
func TestFillTrackIdentityConcurrent(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	errs := make([]error, 2)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, MusicBrainzID: ids[i], OnlyFillMusicBrainzID: true})
		}(i)
	}
	close(start)
	wg.Wait()
	winner := 0
	if errs[0] != nil {
		winner = 1
	}
	require.NoError(t, errs[winner])
	require.ErrorIs(t, errs[1-winner], db.ErrTrackIdentityConflict)
	track, err := s.GetTrack(ctx, db.GetTrackOpts{ID: 1})
	require.NoError(t, err)
	require.Equal(t, ids[winner], *track.MbzID)
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, MusicBrainzID: ids[winner], OnlyFillMusicBrainzID: true}))
	require.ErrorIs(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 999, MusicBrainzID: ids[0], OnlyFillMusicBrainzID: true}), db.ErrNotFound)
}

// TestFillTrackIdentityRollback preserves identity when a later metadata mutation fails.
func TestFillTrackIdentityRollback(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	id := uuid.New()
	err := s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, MusicBrainzID: id, OnlyFillMusicBrainzID: true, AddArtists: []int32{999}})
	require.ErrorIs(t, err, db.ErrNotFound)
	track, err := s.GetTrack(ctx, db.GetTrackOpts{ID: 1})
	require.NoError(t, err)
	require.Nil(t, track.MbzID)
	require.NoError(t, s.Exec(`CREATE TRIGGER reject_track_identity BEFORE UPDATE OF musicbrainz_id ON tracks BEGIN SELECT RAISE(ABORT,'injected'); END`))
	require.Error(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, MusicBrainzID: id, OnlyFillMusicBrainzID: true}))
	track, err = s.GetTrack(ctx, db.GetTrackOpts{ID: 1})
	require.NoError(t, err)
	require.Nil(t, track.MbzID)
}
