package engine

// Import recovery tests (ISSUE-064). The import contract is non-transactional:
// each listen is committed on its own, so a failure or an interruption keeps
// what was already saved, and importing the same file again resumes without
// duplicating listens. A malformed (truncated) file is rejected before any
// listen is written.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

const recoveryProfile int32 = 2

// spotifyExport builds a Spotify history with n distinct listens.
func spotifyExport(t *testing.T, n int) string {
	t.Helper()
	base := time.Date(2023, 11, 14, 22, 0, 0, 0, time.UTC)
	rows := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf(`{"ts":%q,"master_metadata_track_name":"Track %d","master_metadata_album_artist_name":"Fixture","reason_end":"trackdone"}`,
			base.Add(time.Duration(i)*5*time.Minute).Format(time.RFC3339), i%6)
	}
	return "[" + strings.Join(rows, ",") + "]"
}

func writeImport(t *testing.T, content string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "Streaming_History_Audio.json")
	require.NoError(t, os.WriteFile(file, []byte(content), 0600))
	return file
}

// failingStore commits `limit` listens and then fails every write, as a full or
// failing disk would; onLimit optionally models a shutdown instead.
type failingStore struct {
	*sqlite.Sqlite
	limit   int
	saved   *int
	onLimit func()
}

func (s failingStore) SaveListenWithResult(ctx context.Context, o db.SaveListenOpts) (bool, error) {
	if *s.saved >= s.limit {
		if s.onLimit != nil {
			s.onLimit()
			return false, ctx.Err()
		}
		return false, fmt.Errorf("write listen: %w", syscall.ENOSPC)
	}
	inserted, err := s.Sqlite.SaveListenWithResult(ctx, o)
	if err == nil {
		*s.saved++
	}
	return inserted, err
}

func profileListens(t *testing.T, a *profileTestApp) int64 {
	t.Helper()
	n, err := a.store.ProfileListenCount(context.Background(), recoveryProfile)
	require.NoError(t, err)
	return n
}

func stagingEntries(t *testing.T) int {
	t.Helper()
	dir := filepath.Join(cfg.ConfigDir(), "import_staging")
	require.NoError(t, os.MkdirAll(dir, 0700))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	return len(entries)
}

// TestImportDiskFailureKeepsSavedListensAndResumes fails the store after seven
// listens, then imports the same file again until it is complete.
func TestImportDiskFailureKeepsSavedListensAndResumes(t *testing.T) {
	a := newProfileTestApp(t)
	file := writeImport(t, spotifyExport(t, 20))
	staged := stagingEntries(t)
	saved := 0
	result, err := importer.Upload(context.Background(), failingStore{Sqlite: a.store, limit: 7, saved: &saved}, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
	require.ErrorIs(t, err, syscall.ENOSPC)
	require.Equal(t, 7, result.Imported)
	require.EqualValues(t, 7, profileListens(t, a), "listens committed before the failure are kept")
	require.Equal(t, staged, stagingEntries(t), "the staged upload is removed after a failure")
	// The disk recovers: importing the same file again completes without duplicates.
	result, err = importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
	require.NoError(t, err)
	require.Equal(t, 13, result.Imported)
	require.Equal(t, 7, result.Duplicates)
	require.EqualValues(t, 20, profileListens(t, a))
	// A third run is a no-op.
	result, err = importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
	require.NoError(t, err)
	require.Zero(t, result.Imported)
	require.Equal(t, 20, result.Duplicates)
	require.EqualValues(t, 20, profileListens(t, a))
}

// TestImportShutdownMidwayResumesWithoutDuplicates cancels the import context
// after twelve listens, as a service stop would, and restarts the import.
func TestImportShutdownMidwayResumesWithoutDuplicates(t *testing.T) {
	a := newProfileTestApp(t)
	file := writeImport(t, spotifyExport(t, 30))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	saved := 0
	_, err := importer.Upload(ctx, failingStore{Sqlite: a.store, limit: 12, saved: &saved, onLimit: cancel}, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 12, profileListens(t, a))
	result, err := importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
	require.NoError(t, err)
	require.Equal(t, 18, result.Imported)
	require.Equal(t, 12, result.Duplicates)
	require.EqualValues(t, 30, profileListens(t, a))
}

// TestImportTruncatedFileStoresNothing cuts the export at several points; each
// truncated file is rejected before any listen is written and the full file
// still imports afterwards.
func TestImportTruncatedFileStoresNothing(t *testing.T) {
	a := newProfileTestApp(t)
	full := spotifyExport(t, 10)
	for _, cut := range []int{1, len(full) / 3, len(full) / 2, len(full) - 1} {
		file := writeImport(t, full[:cut])
		_, err := importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", file)
		require.ErrorIs(t, err, importer.ErrInvalidFile, "cut at %d", cut)
		require.Zero(t, profileListens(t, a), "cut at %d", cut)
	}
	result, err := importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, recoveryProfile, "spotify", writeImport(t, full))
	require.NoError(t, err)
	require.Equal(t, 10, result.Imported)
}
