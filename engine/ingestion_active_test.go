package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"net/http"

	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

// TestActiveIngestImports verifies durable native and ListenBrainz imports before resolution.
func TestActiveIngestImports(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()

	mb := &mbz.MbzErrorCaller{}
	const native = `{"version":"1","listens":[{"listened_at":"2023-11-14T22:13:20Z",
		"track":{"aliases":[{"alias":"Native Track","is_primary":true}]},
		"album":{"aliases":[{"alias":"Native Album","is_primary":true}]},
		"artists":[{"aliases":[{"alias":"Native Artist","is_primary":true}]}]}]}`
	f := filepath.Join(t.TempDir(), "input.json")
	require.NoError(t, os.WriteFile(f, []byte(native), 0600))
	result, err := importer.Upload(ctx, a.store, mb, 2, "koito", f)
	require.NoError(t, err)
	require.True(t, result.Active)
	require.Equal(t, 1, result.Pending)
	require.Zero(t, result.Imported)
	checkpoints, err := a.store.ReadFileReceipts(ctx, 2, "koito", ingest.Digest([]byte(native)), -1)
	require.NoError(t, err)
	require.Len(t, checkpoints, 1)
	require.Equal(t, 0, checkpoints[0].Ordinal)
	require.Zero(t, checkpoints[0].Snapshot.CatalogTrackID)
	found, err := a.store.ResolveNextIngest(ctx, mb)
	require.NoError(t, err)
	require.True(t, found)
	var trackID int32
	require.NoError(t, a.store.QueryRow(`SELECT track_id FROM listens WHERE profile_id=2`).Scan(&trackID))
	require.NotEqual(t, checkpoints[0].Snapshot.CatalogTrackID, trackID)

	const artistMBID = "12345678-1234-4321-abcd-000000000001"
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	entry, err := zw.Create("listens/2023.jsonl")
	require.NoError(t, err)
	_, err = entry.Write([]byte(`{"listened_at":1700000050,"track_metadata":{"artist_name":"Credit","track_name":"LB Track","additional_info":{"duration_played_ms":90500},"mbid_mapping":{"artists":[{"artist_credit_name":"Credit","artist_mbid":"` + artistMBID + `"}]}}}` + "\n"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	f = filepath.Join(t.TempDir(), "input.zip")
	require.NoError(t, os.WriteFile(f, archive.Bytes(), 0600))
	result, err = importer.Upload(ctx, a.store, mb, 2, "listenbrainz", f)
	require.NoError(t, err)
	require.Equal(t, 1, result.Pending)
	checkpoints, err = a.store.ReadFileReceipts(ctx, 2, "listenbrainz", ingest.Digest(archive.Bytes()), -1)
	require.NoError(t, err)
	require.Len(t, checkpoints, 1)
	require.Equal(t, []ingest.ArtistMapping{{Name: "Credit", MBID: artistMBID}}, checkpoints[0].Snapshot.ArtistMbidMappings)
	found, err = a.store.ResolveNextIngest(ctx, mb)
	require.NoError(t, err)
	require.True(t, found)
	n, err := a.store.Count(`SELECT count(*) FROM listens l JOIN occurrence_credits c ON c.occurrence_id=l.track_id
		JOIN artists a ON a.id=c.artist_id WHERE l.profile_id=2 AND l.listened_at=1700000050 AND l.duration_played=90 AND a.musicbrainz_id=?`, artistMBID)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	const emptyArtist = `{"scrobbles":[{"time":1700000000,"track":{"artists":[""],"title":"Track","album":{"albumtitle":"Album"}}}]}`
	f = filepath.Join(t.TempDir(), "input.json")
	require.NoError(t, os.WriteFile(f, []byte(emptyArtist), 0600))
	_, err = importer.Upload(ctx, a.store, mb, 2, "maloja", f)
	require.ErrorIs(t, err, importer.ErrInvalidFile)
	n, err = a.store.Count(`SELECT count(*) FROM ingest_events WHERE file_format='maloja'`)
	require.NoError(t, err)
	require.Zero(t, n)
}

// TestActiveIngestACK checks the protocol boundary and the post-ACK outbox intent.
func TestActiveIngestACK(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	const path = "/apis/listenbrainz/1/submit-listens"
	const valid = `{"listen_type":"import","payload":[{"listened_at":1700000000,"track_metadata":{"artist_name":"Artist","track_name":"Song","additional_info":{"duration_played":"unknown"}}}]}`
	invalid := strings.TrimSuffix(valid, "]}") + `,{"track_metadata":{"track_name":"No artist"}}]}`

	require.NoError(t, a.store.ActivateRelayDelivery(ctx))
	require.NoError(t, a.store.Exec(`INSERT INTO profile_relays(profile_id,url,token)
		VALUES(1,'http://127.0.0.1:1','fixture-token')`))
	approveFixtureRelay(t, a, 1)
	response := a.request("POST", path, invalid, "first-key", false)
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	n, err := a.store.Count(`SELECT count(*) FROM ingest_events`)
	require.NoError(t, err)
	require.Zero(t, n)
	require.NoError(t, a.store.Exec(`CREATE TABLE active_test_fk(id INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED);
		CREATE TRIGGER active_test_commit AFTER INSERT ON ingest_events BEGIN INSERT INTO active_test_fk VALUES(-1); END`))
	response = a.request("POST", path, valid, "first-key", false)
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Equal(t, "1", response.Header().Get("Retry-After"))
	n, err = a.store.Count(`SELECT count(*) FROM ingest_events`)
	require.NoError(t, err)
	require.Zero(t, n)
	require.NoError(t, a.store.Exec(`DROP TRIGGER active_test_commit`))

	response = a.request("POST", path, valid, "first-key", false)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	n, err = a.store.Count(`SELECT count(*) FROM listens`)
	require.NoError(t, err)
	require.Zero(t, n)

	found, err := a.store.ResolveNextIngest(ctx, &mbz.MbzErrorCaller{})
	require.NoError(t, err)
	require.True(t, found)
	// A malformed playback measurement still admits and saves the listen.
	var played *int64
	require.NoError(t, a.store.QueryRow(`SELECT duration_played FROM listens WHERE profile_id=1`).Scan(&played))
	require.Nil(t, played)
	n, err = a.store.Count(`SELECT count(*) FROM outbox o JOIN integrations i ON i.id=o.destination_id AND i.profile_id=o.profile_id
		JOIN ingest_events e ON e.profile_id=o.profile_id AND e.event_id=o.event_id
		WHERE i.id='relay:1' AND i.source_type='listenbrainz-relay' AND o.state='pending' AND o.closed_at IS NULL AND e.state='accepted'`)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

// TestActiveIngestBusy uses a separate SQLite writer against the running test engine.
func TestActiveIngestBusy(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.NewExisting()
	require.NoError(t, err)
	t.Cleanup(func() { store.Close(ctx) })
	profile, err := store.GetListeningProfileByID(ctx, 1)
	require.NoError(t, err)
	const token = "issue081-busy-fixture"
	_, err = store.SaveApiKey(ctx, db.SaveApiKeyOpts{Key: token, ProfileID: profile.ID})
	require.NoError(t, err)

	t.Cleanup(func() {

		require.NoError(t, store.Exec(`DELETE FROM api_keys WHERE key=?`, token))
	})

	dbConn, err := sql.Open("sqlite", filepath.Join(cfg.ConfigDir(), "tunetrail.db"))
	require.NoError(t, err)
	defer dbConn.Close()
	conn, err := dbConn.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
	require.NoError(t, err)
	defer func() { _, err := conn.ExecContext(ctx, "ROLLBACK"); require.NoError(t, err) }()
	// Replay this request through the actual server so authentication and SQLite's BUSY code are exercised.
	request, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"http://127.0.0.1"+cfg.ListenAddr()+"/apis/listenbrainz/1/submit-listens",
		strings.NewReader(listenBody("single", "Busy track", 1700000000)))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Token "+token)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	require.Equal(t, "1", response.Header.Get("Retry-After"))
}

// TestActiveKoitoValidatesBeforeCatalog verifies that an invalid item leaves no catalog writes.
func TestActiveKoitoValidatesBeforeCatalog(t *testing.T) {
	a := newProfileTestApp(t)

	const input = `{"version":"1","listens":[
		{"listened_at":"2023-11-14T22:13:20Z","track":{"aliases":[{"alias":"Track","is_primary":true}]},
		 "album":{"aliases":[{"alias":"Album","is_primary":true}]},"artists":[{"aliases":[{"alias":"Artist","is_primary":true}]}]},
		{"listened_at":"2023-11-14T22:14:00Z","track":{"aliases":[{"alias":"Other","is_primary":true}]},
		 "album":{"aliases":[{"alias":"Album","is_primary":true}]},"artists":[{"aliases":[{"alias":"","is_primary":true}]}]}
	]}`
	file := filepath.Join(t.TempDir(), "input.json")
	require.NoError(t, os.WriteFile(file, []byte(input), 0600))
	_, err := importer.Upload(context.Background(), a.store, &mbz.MbzErrorCaller{}, 2, "koito", file)
	require.ErrorIs(t, err, importer.ErrInvalidFile)
	for _, table := range []string{"artists", "releases", "tracks", "ingest_events", "listens"} {
		n, err := a.store.Count("SELECT count(*) FROM " + table)
		require.NoError(t, err)
		require.Zero(t, n, table)
	}
}

// TestListenAdmissionErrorClasses verifies admission never falls back after a storage error.
func TestListenAdmissionErrorClasses(t *testing.T) {
	for _, tc := range []struct {
		name, setup     string
		status, listens int
	}{
		{"missing delivery policy", `ALTER TABLE relay_delivery_gate RENAME TO fixture_gate`, 503, 0},
		{"generic after admission begins", `CREATE TRIGGER fixture_admission_abort BEFORE INSERT ON ingest_events
			BEGIN SELECT RAISE(ABORT, 'generic fixture error'); END`, 500, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := newProfileTestApp(t)
			require.NoError(t, a.store.Exec(tc.setup))
			out := a.request("POST", "/apis/listenbrainz/1/submit-listens", listenBody("single", "Retry Fixture", 1714564800), "first-key", false)
			require.Equal(t, tc.status, out.Code, out.Body.String())
			if tc.status == http.StatusServiceUnavailable {
				require.Equal(t, "1", out.Header().Get("Retry-After"))
			} else {
				require.Empty(t, out.Header().Get("Retry-After"))
			}
			n, err := a.store.Count(`SELECT count(*) FROM listens`)
			require.NoError(t, err)
			require.Equal(t, tc.listens, n)
		})
	}
}
