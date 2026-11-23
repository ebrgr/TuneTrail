package engine

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOccurrenceStandaloneCountParity reproduces the rehearsal's metadata mix
// in disposable databases with both writers and compares counts per profile.
func TestOccurrenceStandaloneCountParity(t *testing.T) {
	const path = "/apis/listenbrainz/1/submit-listens"
	t.Run("occurrence", func(t *testing.T) {
		app := newProfileTestApp(t)
		submit := func(key, artist, title, album string, stamp int, client string, want int) {
			t.Helper()
			body := fmt.Sprintf(`{"listen_type":"single","payload":[{"listened_at":%d,"track_metadata":{"artist_name":%q,"track_name":%q,"release_name":%q,"additional_info":{"submission_client":%q}}}]}`,
				stamp, artist, title, album, client)
			resp := app.request(http.MethodPost, path, body, key, false)
			require.Equal(t, want, resp.Code, resp.Body.String())
		}
		// Model history already accepted before occurrence admission, not a new replay.
		require.NoError(t, app.store.Exec(`
			INSERT INTO artists(id) VALUES(1);
			INSERT INTO artist_aliases VALUES(1,'History artist','User',1);
			INSERT INTO releases(id) VALUES(1);
			INSERT INTO release_aliases VALUES(1,'History album','User',1);
			INSERT INTO artist_releases VALUES(1,1,1);
			INSERT INTO tracks(id,release_id) VALUES(1,1);
			INSERT INTO track_aliases VALUES(1,'History track','User',1);
			INSERT INTO artist_tracks VALUES(1,1,1);
			WITH RECURSIVE seq(n) AS (VALUES(0) UNION ALL SELECT n+1 FROM seq WHERE n<79)
			INSERT INTO listens(profile_id,track_id,listened_at,client)
			SELECT 1,1,1779999000+n,'history' FROM seq`))
		resolve := func() {
			t.Helper()
			found, err := app.store.ResolveNextIngest(t.Context(), nil)
			require.NoError(t, err)
			require.True(t, found)
		}
		for i := range 2250 {
			artist, title, album := "Pilot artist", fmt.Sprintf("Synthetic track %d", i%20), "Pilot album"
			want := http.StatusOK
			switch i % 11 {
			case 0:
				album = ""
			case 1:
				artist, album, want = "", "", http.StatusBadRequest
			}
			key := []string{"first-key", "phone-key"}[i%2]
			submit(key, artist, title, album, 1780000000+3*i, fmt.Sprintf("client-%d", i%3), want)
			if want == http.StatusOK {
				resolve()
			}
		}
		submit("second-key", "Second artist", "", "", 1780000000, "secondary", http.StatusBadRequest)
		for i := range 6 {
			submit("second-key", "Second artist", "Second track", "", 1780000000+3*i, "secondary", http.StatusOK)
			resolve()
		}
		for profile, want := range map[int]int{1: 2125, 2: 6, 3: 0} {
			count, err := app.store.Count(fmt.Sprintf(`SELECT count(*) FROM listens WHERE profile_id=%d`, profile))
			require.NoError(t, err)
			require.Equal(t, want, count, "profile %d", profile)
		}
		count, err := app.store.Count(`SELECT count(*) FROM listens l JOIN occurrences o ON o.id=l.track_id
				WHERE l.profile_id=1 AND o.release_id IS NULL`)
		require.NoError(t, err)
		require.Equal(t, 205, count)
	})
}
