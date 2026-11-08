package imagecache

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	webp "github.com/SeriousBug/webp-go-pure/std"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// cache033 sets up a new isolated cache without touching the user's files.
func cache033(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, cfg.Load(func(k string) string {
		if k == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"))
}

// png033 returns an asymmetric image used for actual cache publication.
func png033(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 256, 144))))
	return b.Bytes()
}

// TestPipelineCacheCompatibility033 reads a legacy source and shares one derivative among callers.
func TestPipelineCacheCompatibility033(t *testing.T) {
	cache033(t)
	id := uuid.New()
	source := BuildImagePath(id, ImageSizeSource)
	old, err := os.ReadFile("../test_assets/images033/legacy-small.webp")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(source), 0755))
	require.NoError(t, os.WriteFile(source, old, 0600))
	const count = 20
	errs := make(chan error, count)
	var start sync.WaitGroup
	start.Add(1)
	for range count {
		go func() {
			start.Wait()
			info, err := GetImage(id, "128x128.webp")
			if err == nil && info.Mime != "image/webp" {
				err = errors.New("unexpected MIME")
			}
			errs <- err
		}()
	}
	start.Done()
	for range count {
		require.NoError(t, <-errs)
	}
	saved, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, old, saved)
	result, err := os.ReadFile(BuildImagePath(id, ImageSizeSmall))
	require.NoError(t, err)
	decoded, err := webp.DecodeBytes(result)
	require.NoError(t, err)
	require.Equal(t, image.Pt(48, 48), decoded.Bounds().Size())
}

// TestPipelineFailures033 never replaces an existing source after a failed transform.
func TestPipelineFailures033(t *testing.T) {
	cache033(t)
	id := uuid.New()
	require.NoError(t, SaveImage(id, bytes.NewReader(png033(t))))
	source := BuildImagePath(id, ImageSizeSource)
	before, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Error(t, SaveImage(id, bytes.NewReader([]byte("broken"))))
	require.ErrorIs(t, SaveImage(id, io.LimitReader(zero033{}, imagepipeline.MaxBytes+1)), imagepipeline.ErrTooLarge)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, SaveImageContext(ctx, id, bytes.NewReader(png033(t))), context.Canceled)
	after, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, before, after)
	entries, err := os.ReadDir(filepath.Dir(source))
	require.NoError(t, err)
	for _, entry := range entries {
		require.NotContains(t, entry.Name(), ".image-")
	}
	_, err = GetImage(uuid.New(), "64x64.webp")
	require.ErrorIs(t, err, os.ErrNotExist)
}

type zero033 struct{}

// Read supplies an unlimited input without allocating the whole stream.
func (zero033) Read(p []byte) (int, error) { clear(p); return len(p), nil }

// TestIndependentImages033 exercises separate IDs without sharing cache data.
func TestIndependentImages033(t *testing.T) {
	cache033(t)
	data := png033(t)
	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	errs := make(chan error, len(ids))
	for _, id := range ids {
		go func(id uuid.UUID) { errs <- SaveImage(id, bytes.NewReader(data)) }(id)
	}
	for range ids {
		require.NoError(t, <-errs)
	}
	for _, id := range ids {
		_, err := os.Stat(BuildImagePath(id, ImageSizeSource))
		require.NoError(t, err)
	}
}

// TestCanceledPublication033 leaves the prior file intact when cancellation precedes rename.
func TestCanceledPublication033(t *testing.T) {
	cache033(t)
	id := uuid.New()
	require.NoError(t, saveImage(id, ImageSizeSource, bytes.NewReader([]byte("previous"))))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := &pausedReader{started: make(chan struct{}), resume: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- saveImageContext(ctx, id, ImageSizeSource, input) }()
	<-input.started
	cancel()
	close(input.resume)
	require.ErrorIs(t, <-done, context.Canceled)
	got, err := os.ReadFile(BuildImagePath(id, ImageSizeSource))
	require.NoError(t, err)
	require.Equal(t, []byte("previous"), got)
}

type brokenWrite033 struct{}

// Read fails after supplying bytes to simulate a failed temporary-file write path.
func (brokenWrite033) Read(p []byte) (int, error) {
	return copy(p, "partial"), errors.New("injected read failure")
}

// TestPublicationFailure033 never publishes a partial temporary file.
func TestPublicationFailure033(t *testing.T) {
	cache033(t)
	id := uuid.New()
	require.NoError(t, saveImage(id, ImageSizeSource, bytes.NewReader([]byte("previous"))))
	require.Error(t, saveImage(id, ImageSizeSource, brokenWrite033{}))
	source := BuildImagePath(id, ImageSizeSource)
	got, err := os.ReadFile(source)
	require.NoError(t, err)
	require.Equal(t, []byte("previous"), got)
	entries, err := os.ReadDir(filepath.Dir(source))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

// TestPublicationWaitCancellation036 does not need the current holder to release the gate.
func TestPublicationWaitCancellation036(t *testing.T) {
	unlock := LockPublication()
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := LockPublicationContext(ctx); result <- err }()
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("publication wait did not cancel")
	}
}

// TestPublicationReservations037 checks reference counting, idempotence and canceled admission.
func TestPublicationReservations037(t *testing.T) {
	id := uuid.New()
	first, err := ReservePublication(context.Background(), id)
	require.NoError(t, err)
	second, err := ReservePublication(context.Background(), id)
	require.NoError(t, err)
	first()
	first()
	unlock := LockPublication()
	require.True(t, PublicationReservedLocked(id))
	unlock()
	second()
	unlock = LockPublication()
	require.False(t, PublicationReservedLocked(id))
	unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = ReservePublication(ctx, id)
	require.ErrorIs(t, err, context.Canceled)
	_, err = ReservePublication(context.Background(), uuid.Nil)
	require.Error(t, err)
}
