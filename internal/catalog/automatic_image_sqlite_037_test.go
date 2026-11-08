package catalog_test

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/google/uuid"
)

type scopedSQLiteImages037 struct {
	db.ImageStore
	ids map[uuid.UUID]bool
}

// ImageHasAssociation confines collection checks to this test's fictional IDs.
func (s scopedSQLiteImages037) ImageHasAssociation(ctx context.Context, id uuid.UUID) (bool, error) {
	if !s.ids[id] {
		return true, nil
	}
	return s.ImageStore.ImageHasAssociation(ctx, id)
}

// TestAutomaticImageWithSQLite037 exercises the reservation protocol with real artist and album commits.
func TestAutomaticImageWithSQLite037(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	base, err := store.SaveArtist(ctx, db.SaveArtistOpts{Name: "Fixture parent"})
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	if err = png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data.Bytes()) }))
	defer server.Close()
	for _, kind := range []string{"artist", "album"} {
		id := uuid.New()
		idCopy := id
		t.Cleanup(func() { _ = imagecache.DeleteImage(idCopy) })
		scoped := scopedSQLiteImages037{ImageStore: store, ids: map[uuid.UUID]bool{id: true}}
		entered := make(chan struct{})
		commit := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			release, err := imagecache.ReservePublication(ctx, id)
			if err != nil {
				done <- err
				return
			}
			defer release()
			if err = imagecache.DownloadImageContext(ctx, id, server.URL); err != nil {
				done <- err
				return
			}
			close(entered)
			<-commit
			if kind == "artist" {
				_, err = store.SaveArtist(ctx, db.SaveArtistOpts{Name: "Fictional artist", Image: id, ImageSrc: server.URL})
			} else {
				_, err = store.SaveAlbum(ctx, db.SaveAlbumOpts{Title: "Fictional album", ArtistIDs: []int32{base.ID}, Image: id, ImageSrc: server.URL})
			}
			done <- err
		}()
		select {
		case <-entered:
		case err := <-done:
			t.Fatalf("%s producer failed: %v", kind, err)
		case <-time.After(5 * time.Second):
			t.Fatal("timeout")
		}
		if err := catalog.PruneOrphanedImages(ctx, scoped); err != nil {
			t.Fatal(err)
		}
		path := imagecache.BuildImagePath(id, imagecache.ImageSizeSource)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s removed before commit: %v", kind, err)
		}
		close(commit)
		if err := <-done; err != nil {
			t.Fatal(err)
		}
		associated, err := store.ImageHasAssociation(ctx, id)
		if err != nil || !associated {
			t.Fatalf("%s association %v: %v", kind, associated, err)
		}
		if err := catalog.PruneOrphanedImages(ctx, scoped); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s removed after commit: %v", kind, err)
		}
	}
}

// TestArtistPartialCommit037 preserves an image when an alias error follows commit.
func TestArtistPartialCommit037(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	var data bytes.Buffer
	if err = png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data.Bytes()) }))
	defer server.Close()
	id := uuid.New()
	t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	release, err := imagecache.ReservePublication(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = imagecache.DownloadImageContext(ctx, id, server.URL); err != nil {
		release()
		t.Fatal(err)
	}
	_, err = store.SaveArtist(ctx, db.SaveArtistOpts{Name: "Partial commit", Aliases: []string{""}, Image: id, ImageSrc: server.URL})
	release()
	if err == nil {
		t.Fatal("expected post-commit alias error")
	}
	associated, err := store.ImageHasAssociation(ctx, id)
	if err != nil || !associated {
		t.Fatalf("committed association %v: %v", associated, err)
	}
	scoped := scopedSQLiteImages037{ImageStore: store, ids: map[uuid.UUID]bool{id: true}}
	if err = catalog.PruneOrphanedImages(ctx, scoped); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(imagecache.BuildImagePath(id, imagecache.ImageSizeSource)); err != nil {
		t.Fatalf("committed image removed: %v", err)
	}
}
