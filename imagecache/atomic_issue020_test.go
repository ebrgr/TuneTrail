package imagecache

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type pausedReader struct {
	started, resume chan struct{}
	step            int
}

// Read pauses after one chunk so the published path can be inspected mid-write.
func (r *pausedReader) Read(p []byte) (int, error) {
	if r.step == 0 {
		r.step++
		return copy(p, "new-"), nil
	}
	if r.step == 1 {
		r.step++
		close(r.started)
		<-r.resume
		return copy(p, "image"), nil
	}
	return 0, io.EOF
}

// TestImagePublicationDoesNotExposePartialWrites protects concurrent readers.
func TestImagePublicationDoesNotExposePartialWrites(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, cfg.Load(func(k string) string {
		if k == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"))
	id := uuid.New()
	require.NoError(t, saveImage(id, ImageSizeSource, strings.NewReader("old-image")))
	r := &pausedReader{started: make(chan struct{}), resume: make(chan struct{})}
	var release sync.Once
	defer release.Do(func() { close(r.resume) })
	done := make(chan error, 1)
	go func() { done <- saveImage(id, ImageSizeSource, r) }()
	select {
	case <-r.started:
	case <-time.After(5 * time.Second):
		t.Fatal("image write did not start")
	}
	original, err := os.ReadFile(BuildImagePath(id, ImageSizeSource))
	require.NoError(t, err)
	require.Equal(t, "old-image", string(original))
	release.Do(func() { close(r.resume) })
	require.NoError(t, <-done)
	result, err := os.ReadFile(BuildImagePath(id, ImageSizeSource))
	require.NoError(t, err)
	require.Equal(t, "new-image", string(result))
}

// TestCanceledImageDownloadLeavesNoPublishedFile checks the actual cache entry point.
func TestCanceledImageDownloadLeavesNoPublishedFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, cfg.Load(func(k string) string {
		if k == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"))
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	id := uuid.New()
	done := make(chan error, 1)
	go func() { done <- DownloadImageContext(ctx, id, server.URL) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("image request did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("download did not stop")
	}
	_, err := os.Stat(BuildImagePath(id, ImageSizeSource))
	require.True(t, os.IsNotExist(err))
}
