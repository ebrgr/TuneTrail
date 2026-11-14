package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"github.com/ebrgr/TuneTrail/internal/export"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

func TestPreparedUploadsCountAndScope(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	dir := t.TempDir()
	mb := &mbz.MbzErrorCaller{}
	var z bytes.Buffer
	zw := zip.NewWriter(&z)
	member, err := zw.Create("listens/data.jsonl")
	require.NoError(t, err)
	_, err = member.Write([]byte(`{"listened_at":1700000000,"track_metadata":{"artist_name":"Fictional Artist LB","track_name":"Fictional Track LB","release_name":"Fictional Album LB"}}` + "\n"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	cases := map[string][]byte{
		"spotify":      []byte(`[{"ts":"2023-11-14T22:13:20Z","master_metadata_track_name":"Fictional Track S","master_metadata_album_artist_name":"Fictional Artist S","master_metadata_album_album_name":"Fictional Album S","reason_end":"trackdone","ms_played":100000},{"reason_end":"skip"}]`),
		"maloja":       []byte(`{"scrobbles":[{"time":1700000000,"track":{"artists":["Fictional Artist M"],"title":"Fictional Track M","album":{"albumtitle":"Fictional Album M"}}}]}`),
		"lastfm":       []byte(`[{"track":[{"artist":{"#text":"Fictional Artist F"},"album":{"#text":"Fictional Album F"},"name":"Fictional Track F","date":{"uts":"1700000000"}}]}]`),
		"koito":        []byte(`{"version":"1","user":"unrelated","listens":[{"listened_at":"2023-11-14T22:13:20Z","track":{"aliases":[{"alias":"Fictional Track K","is_primary":true}]},"album":{"aliases":[{"alias":"Fictional Album K","is_primary":true}]},"artists":[{"aliases":[{"alias":"Fictional Artist K","is_primary":true}]}]}]}`),
		"listenbrainz": z.Bytes(),
		// ISSUE-111: TuneTrail exports keep the Koito JSON schema under the new name.
		"tunetrail": []byte(`{"version":"1","user":"unrelated","listens":[{"listened_at":"2023-11-14T22:13:20Z","track":{"aliases":[{"alias":"Fictional Track T","is_primary":true}]},"album":{"aliases":[{"alias":"Fictional Album T","is_primary":true}]},"artists":[{"aliases":[{"alias":"Fictional Artist T","is_primary":true}]}]}]}`),
	}
	for format, content := range cases {
		t.Run(format, func(t *testing.T) {
			f := filepath.Join(dir, format)
			require.NoError(t, os.WriteFile(f, content, 0600))
			result, err := importer.Upload(ctx, a.store, mb, 2, format, f)
			require.NoError(t, err)
			require.Equal(t, 1, result.Imported)
			require.Zero(t, result.Duplicates)
			if format == "spotify" {
				require.Equal(t, 1, result.Ignored)
			}
			result, err = importer.Upload(ctx, a.store, mb, 2, format, f)
			require.NoError(t, err)
			require.Zero(t, result.Imported)
			require.Equal(t, 1, result.Duplicates)
			require.Zero(t, result.Errors)
		})
	}
	// More than one export page, including negative timestamps and same-time distinct tracks.
	require.NoError(t, a.store.Exec(`INSERT INTO listens(track_id,listened_at,profile_id,client) SELECT id,-100,2,'test' FROM tracks LIMIT 2`))
	require.NoError(t, a.store.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<1005) INSERT INTO listens(track_id,listened_at,profile_id,client) SELECT (SELECT min(id) FROM tracks),x,2,'test' FROM n`))
	profile, err := a.store.GetListeningProfileByID(ctx, 2)
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, export.ExportData(ctx, profile, a.store, &output))
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(output.Bytes(), &doc))
	require.Len(t, doc.Listens, 1013) // six formats (tunetrail added in ISSUE-111) plus 1007 inserted
	require.Equal(t, int64(-100), doc.Listens[0].ListenedAt.Unix())
	require.Equal(t, int64(-100), doc.Listens[1].ListenedAt.Unix())
	count, err := a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.EqualValues(t, 1013, count)
	count, err = a.store.ProfileListenCount(ctx, 3)
	require.NoError(t, err)
	require.Zero(t, count)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	f := filepath.Join(dir, "cancelled")
	require.NoError(t, os.WriteFile(f, []byte(`[]`), 0600))
	result, err := importer.Upload(cancelled, a.store, mb, 2, "spotify", f)
	require.Error(t, err)
	require.Equal(t, 1, result.Errors)
	// Malformed JSON never becomes a pending file in the boot import directory.
	f = filepath.Join(dir, "bad")
	require.NoError(t, os.WriteFile(f, []byte(`[] {}`), 0600))
	result, err = importer.Upload(ctx, a.store, mb, 2, "spotify", f)
	require.Error(t, err)
	require.Equal(t, 1, result.Errors)
}

func TestUploadLimitsRejectAndDoNotQueue(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	dir := t.TempDir()
	mb := &mbz.MbzErrorCaller{}
	write := func(name string, b []byte) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, b, 0600))
		return p
	}
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	entry, err := zw.Create("listens/long.jsonl")
	require.NoError(t, err)
	_, err = entry.Write(bytes.Repeat([]byte("x"), (1<<20)+1))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	result, err := importer.Upload(ctx, a.store, mb, 2, "listenbrainz", write("long.zip", archive.Bytes()))
	require.Error(t, err)
	require.Equal(t, 1, result.Errors)
	var empty bytes.Buffer
	zw = zip.NewWriter(&empty)
	require.NoError(t, zw.Close())
	result, err = importer.Upload(ctx, a.store, mb, 2, "listenbrainz", write("empty.zip", empty.Bytes()))
	require.Error(t, err)
	require.Equal(t, 1, result.Errors)
	dense := append([]byte("["), bytes.Repeat([]byte("{},"), 250001)...)
	dense = append(dense, []byte("{}]")...)
	result, err = importer.Upload(ctx, a.store, mb, 2, "spotify", write("dense.json", dense))
	require.ErrorIs(t, err, importer.ErrInvalidFile)
	require.Zero(t, result.Imported)
	count, err := a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestUploadArchiveFailureReportsPartial(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	require.NoError(t, a.store.Exec(`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES (99,'archive-fail','Archive failure',123)`))
	dir := filepath.Join(cfg.ConfigDir(), "import_complete")
	require.NoError(t, os.MkdirAll(dir, 0700))
	block := filepath.Join(dir, "99")
	require.NoError(t, os.WriteFile(block, []byte("block"), 0600))
	t.Cleanup(func() { _ = os.Remove(block) })
	input := filepath.Join(t.TempDir(), "fixture.json")
	require.NoError(t, os.WriteFile(input, []byte(`[{"ts":"2023-11-14T22:13:20Z","master_metadata_track_name":"Archive Track","master_metadata_album_artist_name":"Archive Artist","reason_end":"trackdone"}]`), 0600))
	result, err := importer.Upload(ctx, a.store, &mbz.MbzErrorCaller{}, 99, "spotify", input)
	require.Error(t, err)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 1, result.Errors)
	count, err := a.store.ProfileListenCount(ctx, 99)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

func TestManualImportFailureLeavesBootQueue(t *testing.T) {
	a := newProfileTestApp(t)
	dir := filepath.Join(cfg.ConfigDir(), "import", "2")
	require.NoError(t, os.MkdirAll(dir, 0700))
	source := filepath.Join(dir, "koito_invalid_manual.json")
	require.NoError(t, os.WriteFile(source, []byte(`{"version":"1","listens":[{"artists":[]}]}`), 0600))
	defer os.Remove(source)
	w := a.request("POST", "/apis/web/v1/profiles/second/import", `{"filename":"koito_invalid_manual.json"}`, "", true)
	require.Equal(t, 400, w.Code)
	_, err := os.Stat(source)
	require.True(t, os.IsNotExist(err))
	failedDir := filepath.Join(cfg.ConfigDir(), "import_failed", "2")
	entries, err := os.ReadDir(failedDir)
	require.NoError(t, err)
	found := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-koito_invalid_manual.json") {
			found = true
			t.Cleanup(func() { _ = os.Remove(filepath.Join(failedDir, entry.Name())) })
		}
	}
	require.True(t, found)
}
