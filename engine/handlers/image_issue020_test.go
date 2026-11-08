package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagework"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"time"
)

type missingImageSource struct{ db.ImageStore }

// GetImageSource supplies an invalid fictional URL without making a network call.
func (missingImageSource) GetImageSource(context.Context, uuid.UUID) (string, error) {
	return "://invalid", nil
}

type countedHeader struct {
	*httptest.ResponseRecorder
	calls int
}

// WriteHeader records duplicate attempts to write the same response.
func (w *countedHeader) WriteHeader(status int) { w.calls++; w.ResponseRecorder.WriteHeader(status) }

// TestImageFallbackWritesOneResponse ensures a download error does not continue as success.
func TestImageFallbackWritesOneResponse(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, cfg.Load(func(k string) string {
		if k == cfg.CONFIG_DIR_ENV {
			return dir
		}
		return ""
	}, "test"))
	logger := zerolog.New(io.Discard)
	w := &countedHeader{ResponseRecorder: httptest.NewRecorder()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	image, err := imageHandlerRedownload(w, req, &logger, missingImageSource{}, imagework.New(5, 30*time.Second), uuid.New(), "64x64.webp")
	require.Error(t, err)
	require.Nil(t, image)
	require.Equal(t, 1, w.calls)
	require.Equal(t, 200, w.Code)
	require.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
}
