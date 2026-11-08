package catalog

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
)

type associationStore037 struct {
	db.ImageStore
	mu      sync.Mutex
	managed map[uuid.UUID]bool
	checked map[uuid.UUID]int
}

// ImageHasAssociation keeps unrelated test files outside the collector's scope.
func (s *associationStore037) ImageHasAssociation(_ context.Context, id uuid.UUID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checked[id]++
	associated, managed := s.managed[id]
	if !managed {
		return true, nil
	}
	return associated, nil
}

// setAssociated records a simulated committed catalog reference.
func (s *associationStore037) setAssociated(id uuid.UUID, value bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.managed[id] = value
}

// png037 creates a fictional image for loopback providers.
func png037(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// await037 bounds coordinated waits without adding race-prone sleeps.
func await037(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", name)
	}
}

// TestAutomaticImagesSurviveCollection037 reproduces collection between publish
// and association for artist and album producers, including concurrent downloads.
func TestAutomaticImagesSurviveCollection037(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	orphan := uuid.New()
	store := &associationStore037{managed: map[uuid.UUID]bool{ids[0]: false, ids[1]: false, orphan: false}, checked: make(map[uuid.UUID]int)}
	for _, id := range append(ids, orphan) {
		id := id
		t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	}
	orphanPath := imagecache.BuildImagePath(orphan, imagecache.ImageSizeSource)
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0600); err != nil {
		t.Fatal(err)
	}
	data := png037(t)
	entered := make(chan struct{}, 2)
	releaseDownloads := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entered <- struct{}{}
		select {
		case <-releaseDownloads:
			_, _ = w.Write(data)
		case <-r.Context().Done():
		}
	}))
	var downloadsOnce, commitOnce sync.Once
	associationStarted := make(chan struct{}, 2)
	commit := make(chan struct{})
	defer func() {
		downloadsOnce.Do(func() { close(releaseDownloads) })
		commitOnce.Do(func() { close(commit) })
		server.Close()
	}()
	results := make(chan error, 2)
	for _, id := range ids {
		id := id
		go func() {
			_, err := saveWithAutomaticImage(context.Background(), id, server.URL, false, func() (int, error) {
				associationStarted <- struct{}{}
				<-commit
				store.setAssociated(id, true)
				return 1, nil
			})
			results <- err
		}()
	}
	await037(t, entered, "artist download")
	await037(t, entered, "album download")
	// Both downloads are waiting: the collector must still acquire its gate.
	collected := make(chan error, 1)
	go func() { collected <- PruneOrphanedImages(context.Background(), store) }()
	select {
	case err := <-collected:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("collector blocked by download")
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("unrelated orphan remains: %v", err)
	}
	downloadsOnce.Do(func() { close(releaseDownloads) })
	await037(t, associationStarted, "artist save")
	await037(t, associationStarted, "album save")
	if err := PruneOrphanedImages(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if _, err := os.Stat(imagecache.BuildImagePath(id, imagecache.ImageSizeSource)); err != nil {
			t.Fatalf("reserved source removed: %v", err)
		}
	}
	store.mu.Lock()
	for _, id := range ids {
		if store.checked[id] != 0 {
			store.mu.Unlock()
			t.Fatalf("reserved source checked: %s", id)
		}
	}
	store.mu.Unlock()
	commitOnce.Do(func() { close(commit) })
	for range ids {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if err := PruneOrphanedImages(context.Background(), store); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if _, err := os.Stat(imagecache.BuildImagePath(id, imagecache.ImageSizeSource)); err != nil {
			t.Fatalf("associated source removed: %v", err)
		}
	}
}

// TestFailedSaveDoesNotDeletePotentialAssociation037 leaves uncertain commits to collection.
func TestFailedSaveDoesNotDeletePotentialAssociation037(t *testing.T) {
	data := png037(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	store := &associationStore037{managed: make(map[uuid.UUID]bool), checked: make(map[uuid.UUID]int)}
	injected := errors.New("save failed")
	for _, committed := range []bool{true, false} {
		id := uuid.New()
		idCopy := id
		t.Cleanup(func() { _ = imagecache.DeleteImage(idCopy) })
		_, err := saveWithAutomaticImage(context.Background(), id, server.URL, false, func() (int, error) { store.setAssociated(id, committed); return 0, injected })
		if !errors.Is(err, injected) {
			t.Fatalf("save error: %v", err)
		}
		path := imagecache.BuildImagePath(id, imagecache.ImageSizeSource)
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("premature cleanup: %v", err)
		}
		if err = PruneOrphanedImages(context.Background(), store); err != nil {
			t.Fatal(err)
		}
		_, err = os.Stat(path)
		if committed && err != nil || !committed && !os.IsNotExist(err) {
			t.Fatalf("committed=%v: %v", committed, err)
		}
	}
}

// TestAutomaticImageCancellationAndSkip037 checks no publication or save on cancel,
// plus existing SkipCacheImage and remote-retry behavior.
func TestAutomaticImageCancellationAndSkip037(t *testing.T) {
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
	defer server.Close()
	id := uuid.New()
	t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	ctx, cancel := context.WithCancel(context.Background())
	called := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		_, err := saveWithAutomaticImage(ctx, id, server.URL, false, func() (int, error) { called <- struct{}{}; return 1, nil })
		done <- err
	}()
	await037(t, entered, "download")
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancel stuck")
	}
	select {
	case <-called:
		t.Fatal("save called after cancellation")
	default:
	}
	if _, err := os.Stat(imagecache.BuildImagePath(id, imagecache.ImageSizeSource)); !os.IsNotExist(err) {
		t.Fatalf("canceled file: %v", err)
	}
	calledSkip := false
	v, err := saveWithAutomaticImage(context.Background(), uuid.New(), "http://invalid", true, func() (int, error) { calledSkip = true; return 42, nil })
	if err != nil || !calledSkip || v != 42 {
		t.Fatalf("skip: %d %v", v, err)
	}
	calledRetry := false
	v, err = saveWithAutomaticImage(context.Background(), uuid.New(), "://invalid", false, func() (int, error) { calledRetry = true; return 7, nil })
	if err != nil || !calledRetry || v != 7 {
		t.Fatalf("failed download must allow source association: %d %v", v, err)
	}
}

// TestHistoricalRaceWithoutReservation037 demonstrates the original vulnerable ordering.
func TestHistoricalRaceWithoutReservation037(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	store := &associationStore037{managed: map[uuid.UUID]bool{id: false}, checked: make(map[uuid.UUID]int)}
	data := png037(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	if err := imagecache.DownloadImageContext(ctx, id, server.URL); err != nil {
		t.Fatal(err)
	}
	path := imagecache.BuildImagePath(id, imagecache.ImageSizeSource)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if err := PruneOrphanedImages(ctx, store); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("historical sequence should lose the unassociated file: %v", err)
	}
}

// TestCanceledCollectorWait037 exits without waiting for a held publication gate.
func TestCanceledCollectorWait037(t *testing.T) {
	unlock := imagecache.LockPublication()
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- PruneOrphanedImages(ctx, &associationStore037{managed: map[uuid.UUID]bool{}, checked: map[uuid.UUID]int{}})
	}()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("collector wait did not cancel")
	}
}
