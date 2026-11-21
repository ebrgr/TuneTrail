package engine

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/mbz"
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
	for _, tc := range []struct {
		name   string
		active bool
	}{{"legacy writer", false}, {"active resolver", true}} {
		t.Run(tc.name, func(t *testing.T) {
			a := newProfileTestApp(t)
			require.NoError(t, a.store.Exec(`INSERT INTO artists(id,musicbrainz_id) VALUES(9001,?);
				INSERT INTO artist_aliases(artist_id,alias,source,is_primary) VALUES(9001,'Canonical Name','Manual',1)`, mbid))
			if tc.active {
				require.NoError(t, a.store.Exec(`INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`))
			}
			response := a.request("POST", path, body, "first-key", false)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			if tc.active {
				found, err := a.store.ResolveNextIngest(context.Background(), &mbz.MbzErrorCaller{})
				require.NoError(t, err)
				require.True(t, found)
			}
			count := func(query string) int {
				t.Helper()
				n, err := a.store.Count(query)
				require.NoError(t, err)
				return n
			}
			require.Equal(t, 1, count(`SELECT count(*) FROM listens`))
			require.Equal(t, 1, count(`SELECT count(*) FROM artists`), "no duplicate artist")
			require.Equal(t, 1, count(`SELECT count(*) FROM artist_tracks WHERE artist_id=9001`))
		})
	}
}
