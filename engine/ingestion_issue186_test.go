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
	for _, mode := range []string{"legacy", "occurrence"} {
		t.Run(mode, func(t *testing.T) {
			app := newProfileTestApp(t)
			submit := func(key, artist, title, album string, stamp int, client string, want int) {
				t.Helper()
				body := fmt.Sprintf(`{"listen_type":"single","payload":[{"listened_at":%d,"track_metadata":{"artist_name":%q,"track_name":%q,"release_name":%q,"additional_info":{"submission_client":%q}}}]}`,
					stamp, artist, title, album, client)
				resp := app.request(http.MethodPost, path, body, key, false)
				require.Equal(t, want, resp.Code, resp.Body.String())
			}
			for i := range 80 {
				submit("first-key", "History artist", "History track", "History album", 1779999000+i, "history", http.StatusOK)
			}
			if mode == "occurrence" {
				require.NoError(t, app.store.Exec(`DELETE FROM ingest_events WHERE mode='shadow' AND event_id IS NULL;
					INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write,review) VALUES(1,1,1,1),(2,1,1,0)`))
			}
			resolve := func() {
				t.Helper()
				if mode == "occurrence" {
					found, err := app.store.ResolveNextIngest(t.Context(), nil)
					require.NoError(t, err)
					require.True(t, found)
				}
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
			if mode == "occurrence" {
				count, err := app.store.Count(`SELECT count(*) FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=1 AND o.release_id IS NULL`)
				require.NoError(t, err)
				require.Equal(t, 205, count)
			} else {
				count, err := app.store.Count(`SELECT count(*) FROM release_aliases WHERE alias LIKE 'Synthetic track %'`)
				require.NoError(t, err)
				require.Equal(t, 20, count)
			}
		})
	}
}
