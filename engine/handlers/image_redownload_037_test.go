package handlers

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagework"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

type sourceStore037 struct {
	db.ImageStore
	source string
}

// GetImageSource returns the fictional loopback source for a missing cache entry.
func (s sourceStore037) GetImageSource(context.Context, uuid.UUID) (string, error) {
	return s.source, nil
}

// TestRedownloadAfterMissingSource037 preserves recovery after an absent source.
func TestRedownloadAfterMissingSource037(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, cfg.Load(func(k string) string {
		if k == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"))
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data.Bytes()) }))
	defer server.Close()
	id := uuid.New()
	t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	logger := zerolog.New(io.Discard)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	info, err := imageHandlerRedownload(w, r, &logger, sourceStore037{source: server.URL}, imagework.New(5, 30*time.Second), id, "64x64.webp")
	require.NoError(t, err)
	require.Equal(t, "image/webp", info.Mime)
	_, err = os.Stat(imagecache.BuildImagePath(id, imagecache.ImageSizeSource))
	require.NoError(t, err)
	_, err = os.Stat(info.Path)
	require.NoError(t, err)
}
