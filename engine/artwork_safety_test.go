package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/stretchr/testify/require"
)

type artworkTransport func(*http.Request) (*http.Response, error)

func (f artworkTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingAlbumStore struct{ admin.ArtworkStore }

func (s failingAlbumStore) UpdateAlbum(context.Context, db.UpdateAlbumOpts) error {
	return errors.New("injected database error")
}
func TestArtworkPublicationAndTrackAlbumIdentity(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	user, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	for _, q := range []string{`INSERT INTO artists(id) VALUES(10)`, `INSERT INTO artist_aliases VALUES(10,'Fake Artist','User',1)`, `INSERT INTO releases(id,various_artists) VALUES(11,0)`, `INSERT INTO release_aliases VALUES(11,'Fake Album','User',1)`, `INSERT INTO artist_releases VALUES(10,11,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(12,11,60)`, `INSERT INTO track_aliases VALUES(12,'Fake Track','User',1)`, `INSERT INTO artist_tracks VALUES(10,12,1)`} {
		require.NoError(t, a.store.Exec(q))
	}
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	old, err := admin.ReplaceArtwork(ctx, a.store, user, "album", 11, "", bytes.NewReader(b.Bytes()), admin.ArtworkClient{})
	require.NoError(t, err)
	oldPath := imagecache.BuildImagePath(old, imagecache.ImageSizeSource)
	require.FileExists(t, oldPath)
	require.NoError(t, a.store.SetProfileAvatar(ctx, 2, ptr(old.String()), nil))
	replacement, err := admin.ReplaceArtwork(ctx, a.store, user, "track", 12, "", bytes.NewReader(b.Bytes()), admin.ArtworkClient{})
	require.NoError(t, err)
	require.NotEqual(t, old, replacement)
	require.FileExists(t, oldPath)
	album, err := a.store.GetAlbum(ctx, db.GetAlbumOpts{ID: 11})
	require.NoError(t, err)
	require.Contains(t, album.Image.Small, replacement.String())
	_, err = admin.ReplaceArtwork(ctx, failingAlbumStore{a.store}, user, "album", 11, "", bytes.NewReader(b.Bytes()), admin.ArtworkClient{})
	require.Error(t, err)
	album, err = a.store.GetAlbum(ctx, db.GetAlbumOpts{ID: 11})
	require.NoError(t, err)
	require.Contains(t, album.Image.Small, replacement.String())
	_, err = admin.ReplaceArtwork(ctx, a.store, user, "album", 11, "", strings.NewReader("bad"), admin.ArtworkClient{})
	require.ErrorIs(t, err, imagepipeline.ErrFormat)
	// Unsupported images must not replace the current album association.
	_, err = admin.ReplaceArtwork(ctx, a.store, user, "album", 11, "", bytes.NewReader([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00")), admin.ArtworkClient{})
	require.ErrorIs(t, err, imagepipeline.ErrFormat)
	album, err = a.store.GetAlbum(ctx, db.GetAlbumOpts{ID: 11})
	require.NoError(t, err)
	require.Contains(t, album.Image.Small, replacement.String())
	calls := 0
	transport := artworkTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"http://user:password@another.invalid"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	_, err = admin.ReplaceArtwork(ctx, a.store, user, "album", 11, "http://localhost/image", nil, admin.ArtworkClient{Transport: transport})
	require.Error(t, err)
	require.Equal(t, 1, calls)
	redirects := 0
	validTransport := artworkTransport(func(r *http.Request) (*http.Response, error) {
		redirects++
		if redirects == 1 {
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"http://localhost/final"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(b.Bytes()))}, nil
	})
	remote, err := admin.ReplaceArtwork(ctx, a.store, user, "album", 11, "http://localhost/start", nil, admin.ArtworkClient{Transport: validTransport})
	require.NoError(t, err)
	require.Equal(t, 2, redirects)
	require.NoError(t, a.store.Exec(`INSERT INTO releases(id,various_artists) VALUES(20,0)`))
	changedDuringDownload := artworkTransport(func(r *http.Request) (*http.Response, error) {
		require.NoError(t, a.store.Exec(`UPDATE tracks SET release_id=20 WHERE id=12`))
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(b.Bytes()))}, nil
	})
	_, err = admin.ReplaceArtwork(ctx, a.store, user, "track", 12, "http://localhost/art", nil, admin.ArtworkClient{Transport: changedDuringDownload})
	require.ErrorIs(t, err, db.ErrConflict)
	album, err = a.store.GetAlbum(ctx, db.GetAlbumOpts{ID: 11})
	require.NoError(t, err)
	require.Contains(t, album.Image.Small, remote.String())
	t.Cleanup(func() { _ = imagecache.DeleteImage(remote) })
	t.Cleanup(func() { _ = imagecache.DeleteImage(old); _ = imagecache.DeleteImage(replacement) })
}
func ptr(s string) *string { return &s }
