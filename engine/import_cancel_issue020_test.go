package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

type pauseAfterSave struct {
	*sqlite.Sqlite
	saved, resume chan struct{}
}

// AdmitOccurrence stops after a durable first page to model a partial import.
func (s pauseAfterSave) AdmitOccurrence(ctx context.Context, batch ingest.Batch) ([]ingest.Result, error) {
	inserted, err := s.Sqlite.AdmitOccurrence(ctx, batch)
	if err == nil {
		close(s.saved)
		<-s.resume
	}
	return inserted, err
}

// TestImportCancellationAfterFirstSaveReportsPartial verifies the documented non-transactional contract.
func TestImportCancellationAfterFirstSaveReportsPartial(t *testing.T) {
	a := newProfileTestApp(t)
	staging := filepath.Join(cfg.ConfigDir(), "import_staging")
	require.NoError(t, os.MkdirAll(staging, 0700))
	before, err := os.ReadDir(staging)
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "fixture.json")
	data := spotifyExport(t, 201)
	require.NoError(t, os.WriteFile(file, []byte(data), 0600))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := pauseAfterSave{Sqlite: a.store, saved: make(chan struct{}), resume: make(chan struct{})}
	type output struct {
		result importer.Result
		err    error
	}
	done := make(chan output, 1)
	go func() {
		result, err := importer.Upload(ctx, store, &mbz.MbzErrorCaller{}, 2, "spotify", file)
		done <- output{result, err}
	}()
	select {
	case <-store.saved:
	case <-time.After(5 * time.Second):
		t.Fatal("import did not reach first save")
	}
	cancel()
	close(store.resume)
	select {
	case out := <-done:
		require.ErrorIs(t, out.err, context.Canceled)
		require.Equal(t, 200, out.result.Pending)
		require.Equal(t, 1, out.result.Errors)
	case <-time.After(time.Second):
		t.Fatal("import did not stop")
	}
	after, err := os.ReadDir(staging)
	require.NoError(t, err)
	require.Equal(t, len(before), len(after), "temporary upload must be removed")
	resolveFixtureHistory(t, a)
	n, err := a.store.ProfileListenCount(context.Background(), 2)
	require.NoError(t, err)
	require.EqualValues(t, 200, n)
	n, err = a.store.ProfileListenCount(context.Background(), 1)
	require.NoError(t, err)
	require.Zero(t, n)
}
