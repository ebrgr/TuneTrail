package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

func TestTrackArtistUpdateUsesOneTransaction(t *testing.T) {
	s := mergeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{2}}))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM artist_tracks WHERE track_id=1 AND artist_id=2`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE release_id=1 AND artist_id=2`))
	require.ErrorIs(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{999}}), db.ErrNotFound)
	require.NoError(t, s.Exec(`INSERT INTO artists(id) VALUES(3)`))
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_album_artist BEFORE INSERT ON artist_releases BEGIN SELECT RAISE(ABORT,'injected'); END`))
	require.Error(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{3}}))
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM artist_tracks WHERE track_id=1 AND artist_id=3`))
}
func TestPrimaryArtistRequiresRelation(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.ErrorIs(t, s.SetPrimaryTrackArtist(ctx, 1, 2, true), db.ErrNotFound)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM artist_tracks WHERE track_id=1 AND artist_id=1 AND is_primary=1`))
	require.ErrorIs(t, s.SetPrimaryAlbumArtist(ctx, 1, 2, true), db.ErrNotFound)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM artist_releases WHERE release_id=1 AND artist_id=1 AND is_primary=1`))
}
func TestAliasDeleteRejectsPrimaryAndMissing(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	for _, fn := range []func(context.Context, int32, string) error{s.DeleteArtistAlias, s.DeleteAlbumAlias, s.DeleteTrackAlias} {
		require.ErrorIs(t, fn(ctx, 1, "One"), db.ErrConflict)
		require.ErrorIs(t, fn(ctx, 1, "missing"), db.ErrNotFound)
		require.ErrorIs(t, fn(ctx, 1, " "), db.ErrInvalidInput)
	}
}

func TestDeleteListenRequiresExactProfileAndRecord(t *testing.T) {
	s := mergeFixture(t)
	ctx := db.WithProfile(context.Background(), 1)
	instant := time.Unix(100, 0)
	require.NoError(t, s.DeleteListen(ctx, 1, instant))
	require.ErrorIs(t, s.DeleteListen(ctx, 1, instant), db.ErrNotFound)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=2 AND track_id=1 AND listened_at=100`))
	require.ErrorIs(t, s.DeleteListen(context.Background(), 1, instant), db.ErrInvalidInput)
}

func TestRemoveArtistDoesNotPurgeUnplayedCatalog(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, AddArtists: []int32{2}}))
	require.NoError(t, s.Exec(`INSERT INTO tracks(id,release_id,duration) VALUES(50,1,0)`))
	require.NoError(t, s.Exec(`INSERT INTO track_aliases VALUES(50,'Unplayed','User',1)`))
	require.ErrorIs(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{1}}), db.ErrConflict)
	require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=50`))
	require.ErrorIs(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 1, RemoveArtists: []int32{2}}), db.ErrNotFound)
}
