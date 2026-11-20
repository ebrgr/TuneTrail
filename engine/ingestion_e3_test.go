package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
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
	require.Equal(t, 1, result.Processed)
	require.Zero(t, result.Imported)
	require.Zero(t, result.Duplicates)
	require.Zero(t, result.Ignored)
	require.Zero(t, result.Errors)
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
	// Replaying the same bytes under a different filename reports a duplicate.
	renamed := filepath.Join(t.TempDir(), "renamed")
	require.NoError(t, os.WriteFile(renamed, data, 0600))
	replay, err := importer.Upload(ctx, app.store, nil, 2, format, renamed)
	require.NoError(t, err)
	require.Equal(t, 1, replay.Processed)
	require.Equal(t, 1, replay.Duplicates)
	require.Zero(t, replay.Imported)
	require.Zero(t, replay.Pending)
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

func TestOccurrenceMaloja(t *testing.T) {
	data := []byte(`{"scrobbles":[{"time":1780000000,"track":{"artists":["First","Second"],"title":"Track","album":{"albumtitle":"Album"}}}]}`)
	occurrenceFileFixture(t, "maloja", "maloja_import", data, func(s ingest.Snapshot) {
		require.Equal(t, []string{"First", "Second"}, s.ArtistNames)
		require.Equal(t, "1780000000", s.OriginalTime)
	})
}

func TestOccurrenceNative(t *testing.T) {
	data := []byte(`{"version":"1","listens":[{"listened_at":"2026-05-29T00:00:00.010Z",
		"track":{"disc_number":2,"track_number":1,"mbid_source":"arbitrary-source","mbid_declared_at":"2026-11-20T00:00:00Z","aliases":[{"alias":"Track","is_primary":true}]},
		"album":{"aliases":[{"alias":"Album","is_primary":true}]},
		"artists":[{"aliases":[{"alias":"First","is_primary":true}]},{"aliases":[{"alias":"Second","is_primary":true}]}]}]}`)
	occurrenceFileFixture(t, "koito", "native_import", data, func(s ingest.Snapshot) {
		require.Zero(t, s.CatalogTrackID)
		require.Empty(t, s.MBIDSource)
		require.Empty(t, s.MBIDDeclaredAt)
		require.Equal(t, 2, s.DiscNumber)
		require.Equal(t, 1, s.TrackNumber)
		require.Equal(t, []string{"First", "Second"}, s.ArtistNames)
		require.Equal(t, "2026-05-29T00:00:00.010Z", s.OriginalTime)
	})
}

// TestOccurrenceHTTPFixture covers both historical operations and ephemeral playing now.
func TestOccurrenceHTTPFixture(t *testing.T) {
	app := newProfileTestApp(t)
	require.NoError(t, app.store.Exec(`INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(2,1,1)`))
	const path = "/apis/listenbrainz/1/submit-listens"
	now := app.request(http.MethodPost, path, listenBody("playing_now", "Now", 1780000000), "second-key", false)
	require.Equal(t, http.StatusOK, now.Code, now.Body.String())
	count, err := app.store.Count(`SELECT count(*) FROM ingest_events`)
	require.NoError(t, err)
	require.Zero(t, count)
	for _, kind := range []string{"single", "import"} {
		resp := app.request(http.MethodPost, path, listenBody(kind, "Track "+kind, 1780000020), "second-key", false)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
		found, err := app.store.ResolveNextIngest(context.Background(), nil)
		require.NoError(t, err)
		require.True(t, found)
	}
	count, err = app.store.Count(`SELECT count(*) FROM listens WHERE origin='listenbrainz_api' AND profile_id=2`)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

// TestOccurrenceHTTPZeroTimestamp keeps estimated time and the missing-time digest across modes.
func TestOccurrenceHTTPZeroTimestamp(t *testing.T) {
	app := newProfileTestApp(t)
	require.NoError(t, app.store.Exec(`INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,0),(2,1,1)`))
	for _, tc := range []struct{ key, variant, stamp string }{
		{"first-key", "v9", "0"}, {"second-key", "occurrence", "0"}, {"second-key", "occurrence", "0.25"},
	} {
		raw := `{"listen_type":"single","payload":[{"listened_at":` + tc.stamp + `,"track_metadata":{"artist_name":"Artist","track_name":"Track","release_name":"Album"}}]}`
		resp := app.request(http.MethodPost, "/apis/listenbrainz/1/submit-listens", raw, tc.key, false)
		require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	}
	count, err := app.store.Count(`SELECT count(*) FROM ingest_events WHERE mode='active' AND resolver_variant IN ('v9','occurrence')
		AND time_precision='estimated' AND original_time IS NULL AND json_extract(original_snapshot,'$.original_time') IS NULL
		AND substr(listened_at,1,4)<>'1970'`)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	count, err = app.store.Count(`SELECT count(DISTINCT snapshot_digest) FROM ingest_events WHERE mode='active'`)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// TestOccurrenceZIPZeroTimestamp leaves zero seconds absent in archive receipts.
func TestOccurrenceZIPZeroTimestamp(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("listens/history.jsonl")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`{"listened_at":0,"track_metadata":{"artist_name":"Artist","track_name":"Song","release_name":"ZIP album"}}` + "\n"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	occurrenceFileFixture(t, "listenbrainz", "listenbrainz_import", archive.Bytes(), func(s ingest.Snapshot) {
		require.Empty(t, s.OriginalTime)
	})
}
