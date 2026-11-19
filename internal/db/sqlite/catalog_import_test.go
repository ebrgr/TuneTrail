package sqlite

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestOwnerImportResolution160 keeps names and aliases useful to the owner without
// letting another destination's future ingestion adopt the imported metadata.
func TestOwnerImportResolution160(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	own := db.WithImportedCatalog(ctx, 7)
	other := db.WithoutProfile(db.WithProfile(ctx, 8))
	same := db.WithoutProfile(db.WithProfile(ctx, 7))
	artistID, albumID, trackID := uuid.New(), uuid.New(), uuid.New()
	artist, err := s.SaveArtist(own, db.SaveArtistOpts{Name: "Preseed Artist", MusicBrainzID: artistID, Aliases: []string{"Artist extra"}})
	require.NoError(t, err)
	album, err := s.SaveAlbum(own, db.SaveAlbumOpts{Title: "Preseed Album", ArtistIDs: []int32{artist.ID}, MusicBrainzID: albumID, Aliases: []string{"Album extra"}})
	require.NoError(t, err)
	track, err := s.SaveTrack(own, db.SaveTrackOpts{Title: "Preseed Track", ArtistIDs: []int32{artist.ID}, AlbumID: album.ID, RecordingMbzID: trackID})
	require.NoError(t, err)
	require.NoError(t, s.SaveTrackAliases(own, track.ID, []string{"Track extra"}, "forged Canonical"))
	for _, p := range []struct {
		table string
		id    int32
	}{{"artists", artist.ID}, {"releases", album.ID}, {"tracks", track.ID}} {
		var value any
		require.NoError(t, s.db.QueryRow(`SELECT musicbrainz_id FROM `+p.table+` WHERE id=?`, p.id).Scan(&value))
		require.Nil(t, value)
	}
	_, err = s.GetArtist(same, db.GetArtistOpts{Name: "Artist extra"})
	require.NoError(t, err)
	_, err = s.GetAlbum(same, db.GetAlbumOpts{ArtistID: artist.ID, Title: album.Title})
	require.NoError(t, err)
	_, err = s.GetTrack(same, db.GetTrackOpts{ArtistIDs: []int32{artist.ID}, ReleaseID: album.ID, Title: "Track extra"})
	require.NoError(t, err)
	for _, scope := range []context.Context{ctx, other} {
		_, err = s.GetArtist(scope, db.GetArtistOpts{Name: artist.Name})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetArtist(scope, db.GetArtistOpts{Name: "Artist extra"})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetAlbum(scope, db.GetAlbumOpts{ArtistID: artist.ID, Title: album.Title})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetAlbum(scope, db.GetAlbumOpts{ArtistID: artist.ID, Titles: []string{album.Title}})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetAlbumWithNoMbzIDByTitles(scope, artist.ID, []string{album.Title})
		require.ErrorIs(t, err, db.ErrNotFound)
		_, err = s.GetTrack(scope, db.GetTrackOpts{ArtistIDs: []int32{artist.ID}, ReleaseID: album.ID, Title: track.Title})
		require.ErrorIs(t, err, db.ErrNotFound)
	}
	_, err = s.GetArtist(other, db.GetArtistOpts{MusicBrainzID: artistID})
	require.ErrorIs(t, err, db.ErrNotFound)
	_, err = s.GetAlbum(other, db.GetAlbumOpts{MusicBrainzID: albumID})
	require.ErrorIs(t, err, db.ErrNotFound)
	_, err = s.GetTrack(other, db.GetTrackOpts{MusicBrainzID: trackID})
	require.ErrorIs(t, err, db.ErrNotFound)
	// Enrichment in the owner's future ingestion cannot promote its identity or aliases globally.
	require.NoError(t, s.UpdateArtist(same, db.UpdateArtistOpts{ID: artist.ID, MusicBrainzID: artistID}))
	require.NoError(t, s.UpdateAlbum(same, db.UpdateAlbumOpts{ID: album.ID, MusicBrainzID: albumID}))
	require.NoError(t, s.UpdateTrack(same, db.UpdateTrackOpts{ID: track.ID, MusicBrainzID: trackID}))
	require.NoError(t, s.SaveArtistAliases(same, artist.ID, []string{"Auto artist"}, "MusicBrainz"))
	require.NoError(t, s.SaveAlbumAliases(same, album.ID, []string{"Auto album"}, "MusicBrainz"))
	require.NoError(t, s.SaveTrackAliases(same, track.ID, []string{"Auto track"}, "MusicBrainz"))
	_, err = s.GetArtist(other, db.GetArtistOpts{Name: "Auto artist"})
	require.ErrorIs(t, err, db.ErrNotFound)
	for _, table := range []string{"artists", "releases", "tracks"} {
		require.Equal(t, 0, scalar(t, s, `SELECT COUNT(*) FROM `+table+` WHERE musicbrainz_id IS NOT NULL`))
	}
	// A legitimate later catalog entry remains free to claim the MBID and name for other profiles.
	legitimate, err := s.SaveArtist(ctx, db.SaveArtistOpts{Name: artist.Name, MusicBrainzID: artistID})
	require.NoError(t, err)
	require.NotEqual(t, artist.ID, legitimate.ID)
	resolved, err := s.GetArtist(other, db.GetArtistOpts{Name: artist.Name})
	require.NoError(t, err)
	require.Equal(t, legitimate.ID, resolved.ID)
	// Root can still validate imported metadata explicitly by ID.
	require.NoError(t, s.UpdateAlbum(db.WithCatalogEditorial(ctx), db.UpdateAlbumOpts{ID: album.ID, MusicBrainzID: albumID}))
	resolvedAlbum, err := s.GetAlbum(other, db.GetAlbumOpts{MusicBrainzID: albumID})
	require.NoError(t, err)
	require.Equal(t, album.ID, resolvedAlbum.ID)
}

// TestImportProfileIdentity160 verifies a replacement account gets a fresh profile identity.
func TestImportProfileIdentity160(t *testing.T) {
	s, err := NewInMemory()
	require.NoError(t, err)
	defer s.db.Close()
	ctx := context.Background()
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	user, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "owner", Password: "test-password"})
	require.NoError(t, err)
	p, err := s.GetListeningProfile(ctx, "user-2")
	require.NoError(t, err)
	require.Equal(t, user.ID, p.UserID)
	artist, err := s.SaveArtist(db.WithImportedCatalog(ctx, p.ID), db.SaveArtistOpts{Name: "Retired name"})
	require.NoError(t, err)
	require.NoError(t, s.DeleteListeningProfile(ctx, p.ID, p.DisplayName, 0))
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "replacement", Password: "test-password"})
	require.NoError(t, err)
	reused, err := s.GetListeningProfile(ctx, "user-2")
	require.NoError(t, err)
	require.NotEqual(t, p.ID, reused.ID)
	_, err = s.GetArtist(db.WithoutProfile(db.WithProfile(ctx, reused.ID)), db.GetArtistOpts{Name: "Retired name"})
	require.ErrorIs(t, err, db.ErrNotFound)
	preserved, err := s.GetArtist(ctx, db.GetArtistOpts{ID: artist.ID})
	require.NoError(t, err)
	require.Equal(t, "Retired name", preserved.Name)
}
