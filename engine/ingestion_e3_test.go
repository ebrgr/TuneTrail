package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// TestOccurrenceNativeAliases preserves secondary names and reports conflicts after resolution.
func TestOccurrenceNativeAliases(t *testing.T) {
	app := newProfileTestApp(t)
	ctx := t.Context()
	require.NoError(t, app.store.Exec(`
		INSERT INTO artists(id) VALUES(100);
		INSERT INTO artist_aliases VALUES(100,'Other Artist','User',1);
		INSERT INTO releases(id) VALUES(100);
		INSERT INTO release_aliases VALUES(100,'Other Album','User',1);
		INSERT INTO tracks(id,release_id) VALUES(100,100);
		INSERT INTO track_aliases VALUES(100,'Other Track','User',1);`))
	body := []byte(`{"version":"1","listens":[{"listened_at":"2026-05-29T00:00:00Z",
		"track":{"aliases":[{"alias":"Track","is_primary":true},{"alias":"Track alt"},{"alias":"Other Track"}]},
		"album":{"aliases":[{"alias":"Album","is_primary":true},{"alias":"Album alt"},{"alias":"Other Album"}]},
		"artists":[{"aliases":[{"alias":"Artist","is_primary":true},{"alias":"Artist alt"},{"alias":"Other Artist"}]}]}]}`)
	path := filepath.Join(t.TempDir(), "fixture")
	require.NoError(t, os.WriteFile(path, body, 0600))
	result, err := importer.Upload(ctx, app.store, nil, 2, "koito", path)
	require.NoError(t, err)
	require.Equal(t, 1, result.Pending)
	progress, err := app.store.ResolveNextIngest(ctx, nil)
	require.NoError(t, err)
	require.True(t, progress)
	for _, query := range []string{
		`SELECT count(*) FROM listens WHERE profile_id=2 AND origin='native_import'`,
		`SELECT count(*) FROM ingest_events WHERE profile_id=2 AND state='accepted'`,
		`SELECT count(*) FROM artist_aliases WHERE artist_id<>100 AND alias='Artist alt' AND is_primary=0`,
		`SELECT count(*) FROM release_aliases WHERE release_id<>100 AND alias='Album alt' AND is_primary=0`,
		`SELECT count(*) FROM track_aliases WHERE track_id<>100 AND alias='Track alt' AND is_primary=0`,
		`SELECT count(*) FROM artist_aliases WHERE artist_id=100 AND alias='Other Artist'`,
		`SELECT count(*) FROM release_aliases WHERE release_id=100 AND alias='Other Album'`,
		`SELECT count(*) FROM track_aliases WHERE track_id=100 AND alias='Other Track'`,
	} {
		count, err := app.store.Count(query)
		require.NoError(t, err)
		require.Equal(t, 1, count, query)
	}
	checkpoints, err := app.store.ReadFileReceipts(ctx, 2, "koito", ingest.Digest(body), -1)
	require.NoError(t, err)
	require.Len(t, checkpoints, 1)
	require.Equal(t, 3, checkpoints[0].IgnoredAliasConflicts)

	replay, err := importer.Upload(ctx, app.store, nil, 2, "koito", path)
	require.NoError(t, err)
	require.Equal(t, 1, replay.Duplicates)
	require.Equal(t, 3, replay.IgnoredAliasConflicts)
	require.Zero(t, replay.Pending)
}

// TestManualListenUsesAcceptedOccurrence checks the API's existing success response on repeats.
func TestManualListenUsesAcceptedOccurrence(t *testing.T) {
	app := newProfileTestApp(t)

	const stamp = 1780000010
	received := app.request(http.MethodPost, "/apis/listenbrainz/1/submit-listens", listenBody("single", "Fixture manual", stamp), "second-key", false)
	require.Equal(t, http.StatusOK, received.Code)
	progress, err := app.store.ResolveNextIngest(t.Context(), nil)
	require.NoError(t, err)
	require.True(t, progress)
	var id int
	require.NoError(t, app.store.QueryRow(`SELECT track_id FROM listens WHERE profile_id=2 AND listened_at=?`, stamp).Scan(&id))
	for _, second := range []int{stamp, stamp + 1, stamp + 1} {
		body := fmt.Sprintf(`{"track_id":%d,"unix":%d,"client":"Fixture manual"}`, id, second)
		response := app.request(http.MethodPost, "/apis/web/v1/profiles/second/listens", body, "", true)
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	}
	count, err := app.store.Count(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=?`, id)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	inserted, err := app.store.Count(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=? AND origin='manual'`, id)
	require.NoError(t, err)
	require.Equal(t, 1, inserted)
	receipts, err := app.store.Count(`SELECT count(*) FROM ingest_events WHERE profile_id=2`)
	require.NoError(t, err)
	require.Equal(t, 1, receipts)
}

// TestOccurrenceNativeAliasBudget keeps the listen and reports the ordered alias suffix.
func TestOccurrenceNativeAliasBudget(t *testing.T) {
	for _, size := range []int{1, 220} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			app := newProfileTestApp(t)

			ctx := t.Context()
			aliases := []string{`{"alias":"Canonical Track","is_primary":true}`}
			for i := range size {
				aliases = append(aliases, fmt.Sprintf(`{"alias":"explicit-secondary-alias-%03d-%s"}`, i, strings.Repeat("x", 38)))
			}
			body := []byte(fmt.Sprintf(`{"version":"1","listens":[{"listened_at":"2026-05-29T00:00:00Z",
				"track":{"aliases":[%s]},"album":{"aliases":[{"alias":"Fixture Album","is_primary":true}]},
				"artists":[{"aliases":[{"alias":"Fixture Artist","is_primary":true}]}]}]}`, strings.Join(aliases, ",")))
			path := filepath.Join(t.TempDir(), "history")
			require.NoError(t, os.WriteFile(path, body, 0600))
			result, err := importer.Upload(ctx, app.store, nil, 2, "koito", path)
			require.NoError(t, err)
			require.Equal(t, 1, result.Pending)
			resolveFixtureHistory(t, app)
			checkpoints, err := app.store.ReadFileReceipts(ctx, 2, "koito", ingest.Digest(body), -1)
			require.NoError(t, err)
			require.Len(t, checkpoints, 1)
			kept := checkpoints[0].Snapshot.NativeAliases.Track
			require.Equal(t, size, len(kept)+result.IgnoredAliasLimit)
			require.Equal(t, result.IgnoredAliasLimit, checkpoints[0].Snapshot.IgnoredAliasLimit)
			for i, alias := range kept {
				require.Equal(t, fmt.Sprintf("explicit-secondary-alias-%03d-%s", i, strings.Repeat("x", 38)), alias)
			}
			count, err := app.store.Count(`SELECT count(*) FROM listens WHERE profile_id=2 AND origin='native_import'`)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			if size == 220 {
				require.Positive(t, result.IgnoredAliasLimit)
				require.NotEmpty(t, kept)
			} else {
				require.Zero(t, result.IgnoredAliasLimit)
			}
		})
	}
}

// TestOccurrenceFileOptionalText exercises the common limits through every file adapter.
func TestOccurrenceFileOptionalText(t *testing.T) {
	long := strings.Repeat("€", 3500)
	for _, tc := range []struct{ format, origin, body string }{
		{"koito", "native_import", fmt.Sprintf(`{"version":"1","listens":[{"client":%q,"listened_at":"2026-05-29T00:00:00Z","track":{"aliases":[{"alias":"Track","is_primary":true}]},"album":{"aliases":[{"alias":"Album","is_primary":true}]},"artists":[{"aliases":[{"alias":"Artist","is_primary":true}]}]}]}`, long)},
		{"listenbrainz", "listenbrainz_import", fmt.Sprintf(`{"listened_at":1780000000,"track_metadata":{"artist_name":"Artist","track_name":"Track","release_name":%q}}`, long)},
		{"lastfm", "lastfm_import", fmt.Sprintf(`[{"track":[{"artist":{"#text":"Artist"},"album":{"#text":%q},"name":"Track","date":{"uts":"1780000000"}}]}]`, long)},
		{"maloja", "maloja_import", fmt.Sprintf(`{"scrobbles":[{"time":1780000000,"track":{"artists":["Artist"],"title":"Track","album":{"albumtitle":%q}}}]}`, long)},
		{"spotify", "spotify_import", fmt.Sprintf(`[{"ts":"2026-05-28T21:46:40Z","master_metadata_track_name":"Track","master_metadata_album_artist_name":"Artist","master_metadata_album_album_name":%q,"reason_end":"trackdone"}]`, long)},
	} {
		t.Run(tc.format, func(t *testing.T) {
			input := []byte(tc.body)
			if tc.format == "listenbrainz" {
				var archive bytes.Buffer
				writer := zip.NewWriter(&archive)
				entry, err := writer.Create("listens/history.jsonl")
				require.NoError(t, err)
				_, err = entry.Write(append(input, '\n'))
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				input = archive.Bytes()
			}
			occurrenceFileFixture(t, tc.format, tc.origin, input, func(s ingest.Snapshot) {
				require.Equal(t, 1, s.TruncatedFields)
				if tc.format == "koito" {
					require.Equal(t, strings.Repeat("€", 85), s.Client)
				} else {
					require.Equal(t, strings.Repeat("€", 341), s.Release)
				}
			})
		})
	}
}

// TestOccurrenceFileRejectsOnlyOversizedItem checks that both neighbors survive.
func TestOccurrenceFileRejectsOnlyOversizedItem(t *testing.T) {
	for _, tc := range []struct {
		format, envelope, item string
	}{
		{"koito", `{"version":"1","listens":[%s]}`, `{"listened_at":"2026-05-29T00:00:00Z","track":{"aliases":[{"alias":%q,"is_primary":true}]},"artists":[{"aliases":[{"alias":%q,"is_primary":true}]}]}`},
		{"listenbrainz", "%s", `{"listened_at":1780000000,"track_metadata":{"track_name":%q,"artist_name":%q}}`},
		{"lastfm", `[{"track":[%s]}]`, `{"name":%q,"artist":{"#text":%q},"date":{"uts":"1780000000"}}`},
		{"maloja", `{"scrobbles":[%s]}`, `{"time":1780000000,"track":{"title":%q,"artists":[%q]}}`},
		{"spotify", "[%s]", `{"ts":"2026-05-29T00:00:00Z","master_metadata_track_name":%q,"master_metadata_album_artist_name":%q,"reason_end":"trackdone"}`},
	} {
		t.Run(tc.format, func(t *testing.T) {
			app := newProfileTestApp(t)
			items := []string{
				fmt.Sprintf(tc.item, "First", "Artist"),
				fmt.Sprintf(tc.item, strings.Repeat("<", 3141), strings.Repeat(">", 3121)),
				fmt.Sprintf(tc.item, "Last", "Artist"),
			}
			data := []byte(fmt.Sprintf(tc.envelope, strings.Join(items, ",")))
			if tc.format == "listenbrainz" {
				var archive bytes.Buffer
				writer := zip.NewWriter(&archive)
				entry, err := writer.Create("listens/history.jsonl")
				require.NoError(t, err)
				_, err = fmt.Fprintln(entry, strings.Join(items, "\n"))
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				data = archive.Bytes()
			}
			path := filepath.Join(t.TempDir(), "history")
			require.NoError(t, os.WriteFile(path, data, 0600))
			result, err := importer.Upload(t.Context(), app.store, nil, 2, tc.format, path)
			require.NoError(t, err)
			require.Equal(t, 3, result.Processed)
			require.Equal(t, 1, result.RejectedOversizedItems)
			require.Equal(t, 2, result.Pending)
			resolveFixtureHistory(t, app)
			count, err := app.store.Count(`SELECT count(*) FROM listens WHERE profile_id=2`)
			require.NoError(t, err)
			require.Equal(t, 2, count)
		})
	}
}

// TestOfflineImportMBIDConflicts preserves existing IDs and records the ignored declarations.
func TestOfflineImportMBIDConflicts(t *testing.T) {
	app := newProfileTestApp(t)
	const artistID = "00000000-0000-4000-8000-000000000001"
	const releaseID = "00000000-0000-4000-8000-000000000002"
	const otherArtist = "00000000-0000-4000-8000-000000000003"
	const otherRelease = "00000000-0000-4000-8000-000000000004"
	upload := func(artist, release, at string) (importer.Result, []byte) {
		t.Helper()
		body := []byte(fmt.Sprintf(`{"version":"1","listens":[{"listened_at":%q,
			"track":{"aliases":[{"alias":"Track","is_primary":true}]},
			"album":{"mbid":%q,"aliases":[{"alias":"Release","is_primary":true}]},
			"artists":[{"mbid":%q,"aliases":[{"alias":"Artist","is_primary":true}]}]}]}`, at, release, artist))
		path := filepath.Join(t.TempDir(), "history")
		require.NoError(t, os.WriteFile(path, body, 0600))
		result, err := importer.Upload(t.Context(), app.store, nil, 2, "koito", path)
		require.NoError(t, err)
		return result, body
	}

	first, body := upload(artistID, releaseID, "2026-05-29T00:00:00Z")
	require.Equal(t, 1, first.Pending)
	resolveFixtureHistory(t, app)
	receipts, err := app.store.ReadFileReceipts(t.Context(), 2, "koito", ingest.Digest(body), -1)
	require.NoError(t, err)
	require.Len(t, receipts, 1)
	require.Equal(t, "koito", receipts[0].Snapshot.MBIDSource)
	require.NotEmpty(t, receipts[0].Snapshot.MBIDDeclaredAt)

	second, _ := upload(otherArtist, otherRelease, "2026-05-29T01:00:00Z")
	require.Equal(t, 1, second.Pending)
	resolveFixtureHistory(t, app)
	replay, _ := upload(otherArtist, otherRelease, "2026-05-29T01:00:00Z")
	require.Equal(t, 2, replay.IgnoredMBIDConflicts)
	require.Equal(t, 1, replay.Duplicates)
	var artist, release string
	require.NoError(t, app.store.QueryRow(`SELECT musicbrainz_id FROM artists`).Scan(&artist))
	require.NoError(t, app.store.QueryRow(`SELECT musicbrainz_id FROM releases`).Scan(&release))
	require.Equal(t, artistID, artist)
	require.Equal(t, releaseID, release)
	count, err := app.store.Count(`SELECT count(*) FROM listens WHERE profile_id=2`)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
