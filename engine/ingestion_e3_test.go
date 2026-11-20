package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// occurrenceFileFixture exercises upload admission and materialization on a disposable database.
func occurrenceFileFixture(t *testing.T, format, origin string, data []byte, verify func(ingest.Snapshot)) {
	t.Helper()
	ctx := context.Background()
	app := newProfileTestApp(t)
	require.NoError(t, app.store.Exec(`INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(2,1,1)`))
	path := filepath.Join(t.TempDir(), "staged")
	require.NoError(t, os.WriteFile(path, data, 0600))
	result, err := importer.Upload(ctx, app.store, nil, 2, format, path)
	require.NoError(t, err)
	require.Equal(t, 1, result.Pending)
	count, err := app.store.Count(`SELECT count(*) FROM tracks`)
	require.NoError(t, err)
	require.Zero(t, count)
	checkpoints, err := app.store.ReadFileReceipts(ctx, 2, format, ingest.Digest(data), -1)
	require.NoError(t, err)
	require.Len(t, checkpoints, 1)
	verify(checkpoints[0].Snapshot)
	require.NoError(t, os.Remove(path))
	found, err := app.store.ResolveNextIngest(ctx, nil)
	require.NoError(t, err)
	require.True(t, found)
	var actual string
	require.NoError(t, app.store.QueryRow(`SELECT origin FROM listens WHERE profile_id=2`).Scan(&actual))
	require.Equal(t, origin, actual)
}

func TestOccurrenceListenBrainzZIP(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("listens/history.jsonl")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`{"listened_at":1780000000.125,"track_metadata":{"artist_name":"Artist","track_name":"Song","release_name":"ZIP album","additional_info":{"discnumber":2,"tracknumber":"3"}}}` + "\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	occurrenceFileFixture(t, "listenbrainz", "listenbrainz_import", archive.Bytes(), func(s ingest.Snapshot) {
		require.Equal(t, "1780000000.125", s.OriginalTime)
		require.Equal(t, 2, s.DiscNumber)
		require.Equal(t, 3, s.TrackNumber)
	})
}

func TestOccurrenceLastFM(t *testing.T) {
	data := []byte(`[{"track":[{"artist":{"#text":"Solo"},"album":{"#text":"Album"},"name":"Track","date":{"uts":"1780000000"}}]}]`)
	occurrenceFileFixture(t, "lastfm", "lastfm_import", data, func(s ingest.Snapshot) {
		require.Equal(t, []string{"Solo"}, s.ArtistNames)
		require.Equal(t, "1780000000", s.OriginalTime)
		require.Empty(t, s.Recording)
	})
}

func TestOccurrenceSpotify(t *testing.T) {
	data := []byte(`[{"ts":"2026-05-28T21:46:40.2500-03:00","master_metadata_track_name":"Track","master_metadata_album_artist_name":"Artist","master_metadata_album_album_name":"Album","reason_end":"trackdone","ms_played":180125}]`)
	occurrenceFileFixture(t, "spotify", "spotify_import", data, func(s ingest.Snapshot) {
		require.Equal(t, "2026-05-28T21:46:40.2500-03:00", s.OriginalTime)
		require.Equal(t, "end", s.TimeSemantics)
		require.EqualValues(t, 180125, s.DurationMS)
	})
}
