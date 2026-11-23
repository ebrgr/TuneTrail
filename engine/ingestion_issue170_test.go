package engine

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMappedArtistMBIDResolution170 checks that an artist MBID that only appears in
// mbid_mapping resolves to the existing artist instead of creating a duplicate.
func TestMappedArtistMBIDResolution170(t *testing.T) {
	const mbid = "11111111-2222-4333-8444-555555555555"
	const path = "/apis/listenbrainz/1/submit-listens"
	// The spelling differs from the stored name, so only the MBID can identify the artist.
	body := fmt.Sprintf(`{"listen_type":"single","payload":[{"listened_at":1700000000,"track_metadata":{
		"artist_name":"Spelled Differently","track_name":"Mapped Song","release_name":"Mapped Album",
		"mbid_mapping":{"artist_mbids":[%q]}}}]}`, mbid)
	for _, useMapping := range []bool{false, true} {
		t.Run(fmt.Sprintf("mapping=%t", useMapping), func(t *testing.T) {
			a := newProfileTestApp(t)
			require.NoError(t, a.store.Exec(`INSERT INTO artists(id,musicbrainz_id) VALUES(9001,?);
				INSERT INTO artist_aliases(artist_id,alias,source,is_primary) VALUES(9001,'Canonical Name','Manual',1)`, mbid))
			payload := body
			if !useMapping {
				payload = strings.Replace(payload, "mbid_mapping", "additional_info", 1)
			}
			response := a.request("POST", path, payload, "first-key", false)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			resolveFixtureHistory(t, a)
			for _, query := range []string{
				`SELECT count(*) FROM listens`,
				`SELECT count(*) FROM artists`,
				`SELECT count(*) FROM artist_tracks WHERE artist_id=9001`,
				`SELECT count(*) FROM artist_aliases WHERE artist_id=9001 AND alias='Spelled Differently'`,
			} {
				n, err := a.store.Count(query)
				require.NoError(t, err)
				require.Equal(t, 1, n, query)
			}
		})
	}
}

// TestOccurrenceArtistMBIDBranches exercises the profile catalog and the name fallback.
func TestOccurrenceArtistMBIDBranches(t *testing.T) {
	const mbid = "abcdefab-2222-4333-8444-555555555555"
	for _, tc := range []struct {
		name      string
		artistIDs string
		duplicate bool
		album     string
		want      int
	}{
		{"ambiguous case variants", mbid, true, "Album", 0},
		{"unknown MBID", "11111111-2222-4333-8444-555555555555", false, "Album", 2},
		{"no MBID", "", false, "Album", 2},
		{"standalone", mbid, false, "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := newProfileTestApp(t)
			require.NoError(t, app.store.Exec(`INSERT INTO artists(id,musicbrainz_id) VALUES(9001,?);
				INSERT INTO artist_aliases VALUES(9001,'Canonical Name','Manual',1)`, mbid))
			if tc.duplicate {
				require.NoError(t, app.store.Exec(`INSERT INTO artists(id,musicbrainz_id) VALUES(9002,?);
					INSERT INTO artist_aliases VALUES(9002,'Other Name','Manual',1)`, strings.ToUpper(mbid)))
			}
			body := fmt.Sprintf(`{"listen_type":"single","payload":[{"listened_at":1700000000,"track_metadata":{
				"artist_name":"Observed Name","track_name":"Mapped Song","release_name":%q,
				"additional_info":{"artist_mbids":[%q]}}}]}`, tc.album, tc.artistIDs)
			res := app.request(http.MethodPost, "/apis/listenbrainz/1/submit-listens", body, "first-key", false)
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
			progress, err := app.store.ResolveNextIngest(t.Context(), nil)
			require.NoError(t, err)
			require.True(t, progress)
			if tc.duplicate {
				count, err := app.store.Count(`SELECT count(*) FROM ingest_events WHERE state='pending_identity'
					AND json_extract(reason_codes,'$[0]')='ambiguous_artist_mbid' AND event_id IS NULL`)
				require.NoError(t, err)
				require.Equal(t, 1, count)
				count, err = app.store.Count(`SELECT count(*) FROM listens`)
				require.NoError(t, err)
				require.Zero(t, count)
				return
			}
			count, err := app.store.Count(`SELECT count(*) FROM artists`)
			require.NoError(t, err)
			require.Equal(t, tc.want, count)
			count, err = app.store.Count(`SELECT count(*) FROM listens`)
			require.NoError(t, err)
			require.Equal(t, 1, count)
			if tc.album == "" {
				count, err = app.store.Count(`SELECT count(*) FROM occurrence_credits WHERE artist_id=9001 AND name='Observed Name'`)
				require.NoError(t, err)
				require.Equal(t, 1, count)
			}
		})
	}
}
