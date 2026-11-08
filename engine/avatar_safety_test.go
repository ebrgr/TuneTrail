package engine

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type pausedAvatarStore struct {
	admin.AvatarStore
	staged chan string
	resume chan struct{}
}

func (s pausedAvatarStore) SetProfileAvatar(ctx context.Context, id int32, img, icon *string) error {
	if img != nil {
		s.staged <- *img
		<-s.resume
	}
	return s.AvatarStore.SetProfileAvatar(ctx, id, img, icon)
}

type failingImageStore struct{ db.ImageStore }

func (s failingImageStore) ImageHasAssociation(context.Context, uuid.UUID) (bool, error) {
	return false, errors.New("injected")
}
func TestAvatarPublicationAndCollectorSafety(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	user, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	store := pausedAvatarStore{a.store, make(chan string, 1), make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, e := admin.SetAvatar(ctx, store, user, 2, "second", "", bytes.NewReader(data.Bytes()))
		done <- e
	}()
	var img string
	select {
	case img = <-store.staged:
	case <-time.After(5 * time.Second):
		t.Fatal("avatar not staged")
	}
	id := uuid.MustParse(img)
	path := imagecache.BuildImagePath(id, imagecache.ImageSizeSource)
	require.FileExists(t, path)
	pruned := make(chan error, 1)
	go func() { pruned <- catalog.PruneOrphanedImages(ctx, a.store) }()
	select {
	case <-pruned:
		t.Fatal("collector ran while avatar was being published")
	case <-time.After(50 * time.Millisecond):
	}
	close(store.resume)
	require.NoError(t, <-done)
	require.NoError(t, <-pruned)
	require.FileExists(t, path)
	// A database error must leave potentially associated images untouched.
	other := uuid.New()
	otherPath := imagecache.BuildImagePath(other, imagecache.ImageSizeSource)
	require.NoError(t, os.MkdirAll(filepath.Dir(otherPath), 0755))
	require.NoError(t, os.WriteFile(otherPath, []byte("fixture"), 0600))
	t.Cleanup(func() { _ = imagecache.DeleteImage(other) })
	require.Error(t, catalog.PruneOrphanedImages(ctx, failingImageStore{a.store}))
	require.FileExists(t, otherPath)
}

// TestUnsupportedAvatar033 preserves the previous profile when the image is rejected.
func TestUnsupportedAvatar033(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	user, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	before, err := a.store.GetListeningProfileByID(ctx, 2)
	require.NoError(t, err)
	_, err = admin.SetAvatar(ctx, a.store, user, 2, "second", "", bytes.NewReader([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00")))
	require.ErrorIs(t, err, imagepipeline.ErrFormat)
	after, err := a.store.GetListeningProfileByID(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, before.AvatarImageID, after.AvatarImageID)
}
