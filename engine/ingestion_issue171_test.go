package engine

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestShadowReceiptACK verifies deferred commit failure, retry and playing_now.
func TestShadowReceiptACK(t *testing.T) {
	a := newProfileTestApp(t)
	const path = "/apis/listenbrainz/1/submit-listens"
	const body = `{"listen_type":"import","payload":[
		{"listened_at":1700000000,"track_metadata":{"artist_name":"Artist","track_name":"Song","additional_info":{"token":"private"}}},
		{"listened_at":1700000000,"track_metadata":{"artist_name":"Artist","track_name":"Song"}}]}`
	require.NoError(t, a.store.Exec(`CREATE TABLE shadow_test_fk(id INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED)`))
	require.NoError(t, a.store.Exec(`CREATE TRIGGER shadow_test_commit AFTER INSERT ON ingest_events BEGIN INSERT INTO shadow_test_fk VALUES(-1); END`))
	count := func(table string, expected int) {
		t.Helper()
		n, err := a.store.Count("SELECT count(*) FROM " + table)
		require.NoError(t, err)
		require.Equal(t, expected, n)
	}
	require.Equal(t, http.StatusServiceUnavailable, a.request("POST", path, body, "first-key", false).Code)
	count("ingest_events", 0)
	count("listens", 0)
	require.NoError(t, a.store.Exec(`DROP TRIGGER shadow_test_commit`))
	response := a.request("POST", path, body, "first-key", false)
	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"status":"ok"}`, response.Body.String())
	count("ingest_events", 2)
	count("ingest_events WHERE duplicate_decision='review'", 1)
	count("ingest_events WHERE original_snapshot LIKE '%private%'", 0)
	count("listens", 1)
	require.Equal(t, http.StatusOK, a.request("POST", path, listenBody("playing_now", "Only memory", 0), "first-key", false).Code)
	count("ingest_events", 2)
	count("listens", 1)
	require.NoError(t, a.store.Exec(`CREATE TRIGGER shadow_test_fail BEFORE INSERT ON ingest_events BEGIN SELECT RAISE(ABORT,'shadow only'); END`))
	require.Equal(t, http.StatusOK, a.request("POST", path, listenBody("single", "Fail open", 1700000001), "first-key", false).Code)
	count("ingest_events", 2)
	count("listens", 2)
}

// TestShadowFileResume covers ignored ordinals, a renamed resend and the staging-independent cursor.
func TestShadowFileResume(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	data := []byte(`[
		{"ts":"2026-11-01T01:30:00-04:00","master_metadata_track_name":"Song","master_metadata_album_artist_name":"Artist","reason_end":"trackdone"},
		{"reason_end":"skip"},
		{"ts":"2026-11-01T01:30:00-04:00","master_metadata_track_name":"Song","master_metadata_album_artist_name":"Artist","reason_end":"trackdone"}
	]`)
	for i, name := range []string{"first.json", "renamed.json"} {
		file := filepath.Join(t.TempDir(), name)
		require.NoError(t, os.WriteFile(file, data, 0600))
		result, err := importer.Upload(ctx, a.store, &mbz.MbzErrorCaller{}, 2, "spotify", file)
		require.NoError(t, err)
		require.Equal(t, 1-i, result.Imported)
		require.Equal(t, 1+i, result.Duplicates)
		require.Equal(t, 1, result.Pending)
		require.Equal(t, 1, result.Ignored)
		require.NoError(t, os.Remove(file))
	}
	checkpoints, err := a.store.ReadFileReceipts(ctx, 2, "spotify", ingest.Digest(data), -1)
	require.NoError(t, err)
	require.Len(t, checkpoints, 2)
	require.Equal(t, 0, checkpoints[0].Ordinal)
	require.Equal(t, 2, checkpoints[1].Ordinal)
	require.True(t, checkpoints[1].Pending)
	tail, err := a.store.ReadFileReceipts(ctx, 2, "spotify", ingest.Digest(data), checkpoints[0].Ordinal)
	require.NoError(t, err)
	require.Equal(t, checkpoints[1:], tail)
	other, err := a.store.ReadFileReceipts(ctx, 1, "spotify", ingest.Digest(data), -1)
	require.NoError(t, err)
	require.Empty(t, other)
}
