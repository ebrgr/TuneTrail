package export_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

type pages struct{ calls int }

func (s *pages) GetExportPage(ctx context.Context, o db.GetExportPageOpts) ([]*db.ExportItem, error) {
	s.calls++
	if s.calls == 1 {
		if !o.FirstPage {
			return nil, errors.New("missing first page")
		}
		return []*db.ExportItem{{TrackID: 1, ListenedAt: time.Unix(-100, 0)}, {TrackID: 2, ListenedAt: time.Unix(-100, 0)}}, nil
	}
	if o.FirstPage || o.TrackID != 2 || o.ListenedAt.Unix() != -100 {
		return nil, errors.New("bad cursor")
	}
	return nil, nil
}

type failWriter struct{ left int }

func (w *failWriter) Write(p []byte) (int, error) {
	if w.left <= 0 {
		return 0, io.ErrClosedPipe
	}
	w.left--
	return len(p), nil
}
func TestExportNegativeDatesAndWriterFailures(t *testing.T) {
	p := &models.ListeningProfile{ID: 1, DisplayName: "Usuário \"a\""}
	var b bytes.Buffer
	require.NoError(t, export.ExportData(context.Background(), p, &pages{}, &b))
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(b.Bytes(), &doc))
	require.Equal(t, p.DisplayName, doc.User)
	require.Len(t, doc.Listens, 2)
	require.Equal(t, int64(-100), doc.Listens[0].ListenedAt.Unix())
	for i := 0; i < 7; i++ {
		err := export.ExportData(context.Background(), p, &pages{}, &failWriter{left: i})
		require.ErrorIs(t, err, io.ErrClosedPipe)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, export.ExportData(ctx, p, &pages{}, io.Discard), context.Canceled)
}

type cancelWriter struct {
	bytes.Buffer
	calls  int
	cancel context.CancelFunc
}

// Write triggers cancellation after the stream has already begun.
func (w *cancelWriter) Write(p []byte) (int, error) {
	n, err := w.Buffer.Write(p)
	w.calls++
	if w.calls == 3 {
		w.cancel()
	}
	return n, err
}

// TestExportCancellationDuringStreamDoesNotPretendCompletion checks partial output.
func TestExportCancellationDuringStreamDoesNotPretendCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &cancelWriter{cancel: cancel}
	err := export.ExportData(ctx, &models.ListeningProfile{ID: 1, DisplayName: "Fixture"}, &pages{}, w)
	require.ErrorIs(t, err, context.Canceled)
	require.NotEmpty(t, w.String())
	require.False(t, json.Valid(w.Bytes()))
}
