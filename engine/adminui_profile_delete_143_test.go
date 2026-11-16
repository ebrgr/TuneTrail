package engine

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"net/url"
	"testing"
)

// deletionFixture143 contains homonyms, case variants, reversed credits and collaborations in two profiles.
func deletionFixture143(t *testing.T) *profileTestApp {
	t.Helper()
	a := taskApp139(t)
	for _, q := range []string{
		`INSERT INTO artists(id) VALUES(30),(40)`,
		`INSERT INTO artist_aliases VALUES(30,'Source Artist','User',1),(40,'source Artist','User',1),(20,'Source Artist','User',0)`,
		`INSERT INTO releases(id,various_artists) VALUES(21,0),(31,0),(41,0),(51,0),(61,0)`,
		`INSERT INTO release_aliases VALUES(21,'Source Album','User',1),(31,'Source Album','User',1),(41,'Source Album','User',1),(51,'source Album','User',1),(61,'Source Album','User',1)`,
		`INSERT INTO artist_releases VALUES(30,21,1),(20,31,1),(10,41,1),(20,41,0),(10,51,1),(10,61,0),(20,61,1)`,
		`INSERT INTO tracks(id,release_id,duration) VALUES(22,21,0),(32,31,0),(42,41,0),(52,51,0),(62,61,0),(72,11,0)`,
		`INSERT INTO track_aliases VALUES(22,'Source Track','User',1),(32,'Source Track','User',1),(42,'Source Track','User',1),(52,'source Track','User',1),(62,'Source Track','User',1),(72,'Source Track','User',1)`,
		`INSERT INTO artist_tracks VALUES(30,22,1),(20,32,1),(10,42,1),(20,42,0),(40,52,1),(10,62,0),(20,62,1),(10,72,1)`,
		`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,101),(2,22,100),(2,32,100),(2,42,100),(2,52,100),(2,62,100),(2,72,100),(3,22,100),(3,42,100)`,
	} {
		require.NoError(t, a.store.Exec(q))
	}
	return a
}

// deleteForm143 extracts the server's real plan, CSRF and required submission ticket.
func deleteForm143(t *testing.T, a *profileTestApp, base string) (url.Values, *httptest.ResponseRecorder) {
	t.Helper()
	w := taskRequest139(a, "GET", base+"/delete", a.cookie, nil, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	v := url.Values{"profile_id": {"2"}, "profile_slug": {"second"}, "confirm": {"1"}}
	for _, name := range []string{"csrf_token", "submission_token", "plan"} {
		v.Set(name, taskField139(t, w, name))
	}
	return v, w
}

// TestProfileDeleteNames143 proves count/write parity and that neither catalog nor other profiles change.
func TestProfileDeleteNames143(t *testing.T) {
	for _, tc := range []struct {
		kind      string
		id, count int
		removed   []int
	}{
		{"track", 12, 4, []int{12, 22, 72}}, {"album", 11, 4, []int{12, 22, 72}},
		{"artist", 10, 4, []int{12, 22, 72}}, {"track", 42, 2, []int{42, 62}}, {"album", 41, 2, []int{42, 62}},
	} {
		t.Run(fmt.Sprintf("%s-%d", tc.kind, tc.id), func(t *testing.T) {
			a := deletionFixture143(t)
			base := fmt.Sprintf("/second/manage/%s/%d", tc.kind, tc.id)
			v, w := deleteForm143(t, a, base)
			require.Contains(t, w.Body.String(), fmt.Sprintf(`data-delete-count="%d"`, tc.count))
			if tc.kind == "artist" {
				require.Contains(t, w.Body.String(), "collaboration")
			}
			post := taskRequest139(a, "POST", base+"/confirm", a.cookie, v, nil)
			require.Equal(t, 303, post.Code, post.Body.String())
			require.Equal(t, "/second", post.Header().Get("Location"))
			require.Equal(t, fmt.Sprint(tc.count), post.Header().Get("X-TuneTrail-Deleted-Listens"))
			for _, id := range []int{12, 22, 32, 42, 52, 62, 72} {
				var count int
				require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=?`, id).Scan(&count))
				expected := 1
				if id == 12 {
					expected = 2
				}
				for _, removed := range tc.removed {
					if id == removed {
						expected = 0
					}
				}
				require.Equal(t, expected, count, fmt.Sprint(id))
			}
			var n int
			require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=3`).Scan(&n))
			require.Equal(t, 3, n)
			for table, expected := range map[string]int{"artists": 4, "releases": 6, "tracks": 8, "artist_aliases": 5, "track_aliases": 8, "release_aliases": 6, "artist_tracks": 10, "artist_releases": 8} {
				require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM `+table).Scan(&n))
				require.Equal(t, expected, n, table)
			}
			require.Equal(t, 409, taskRequest139(a, "POST", base+"/confirm", a.cookie, v, nil).Code)
		})
	}
}

// TestProfileDeleteSecurity143 exercises all authority checks through the production router.
func TestProfileDeleteSecurity143(t *testing.T) {
	for _, violation := range []string{"profile-id", "profile-slug", "route-id", "route-profile", "csrf", "ticket", "visitor", "expired", "origin", "failure", "stale"} {
		t.Run(violation, func(t *testing.T) {
			a := deletionFixture143(t)
			base := "/second/manage/track/12"
			v, _ := deleteForm143(t, a, base)
			cookie := a.cookie
			headers := map[string]string{}
			switch violation {
			case "profile-id":
				v.Set("profile_id", "3")
			case "profile-slug":
				v.Set("profile_slug", "empty")
			case "route-id":
				base = "/second/manage/track/22"
			case "route-profile":
				base = "/empty/manage/track/12"
				v.Set("profile_id", "3")
				v.Set("profile_slug", "empty")
			case "csrf":
				v.Set("csrf_token", "bad")
			case "ticket":
				v.Del("submission_token")
			case "visitor":
				cookie = ""
			case "expired":
				require.NoError(t, a.store.Exec(`UPDATE sessions SET expires_at=1 WHERE id=?`, cookie))
			case "origin":
				headers["Origin"] = "https://foreign.test"
				headers["Sec-Fetch-Site"] = "cross-site"
			case "failure":
				require.NoError(t, a.store.Exec(`CREATE TRIGGER fail_personal_delete AFTER DELETE ON listens WHEN OLD.track_id=22 BEGIN SELECT RAISE(ABORT,'injected midway'); END`))
			case "stale":
				require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,999)`))
			}
			w := taskRequest139(a, "POST", base+"/confirm", cookie, v, headers)
			require.GreaterOrEqual(t, w.Code, 400, w.Body.String())
			var n int
			require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2`).Scan(&n))
			expected := 8
			if violation == "stale" {
				expected++
			}
			require.Equal(t, expected, n)
		})
	}
	a := deletionFixture143(t)
	for _, kind := range []struct {
		name string
		id   int
	}{{"track", 12}, {"album", 11}, {"artist", 10}} {
		base := fmt.Sprintf("/default/manage/%s/%d", kind.name, kind.id)
		w := taskRequest139(a, "GET", base+"/delete", a.cookie, nil, nil)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Contains(t, w.Body.String(), `data-delete-count="0"`)
		v := url.Values{"profile_id": {"1"}, "profile_slug": {"default"}, "confirm": {"1"}}
		for _, key := range []string{"csrf_token", "submission_token", "plan"} {
			v.Set(key, taskField139(t, w, key))
		}
		post := taskRequest139(a, "POST", base+"/confirm", a.cookie, v, map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"})
		require.Equal(t, 204, post.Code, post.Body.String())
		require.Equal(t, "/default", post.Header().Get("X-TuneTrail-Navigate"))
		require.Equal(t, "0", post.Header().Get("X-TuneTrail-Deleted-Listens"))
	}
}

// TestProfileDeleteExactText143 rejects case/space variants, partial credit matches and stale credits.
func TestProfileDeleteExactText143(t *testing.T) {
	for _, tc := range []struct {
		name, kind, change string
		count              int
	}{
		{"track case", "track", `UPDATE track_aliases SET alias='source Track' WHERE track_id=22`, 3},
		{"track spaces", "track", `UPDATE track_aliases SET alias='Source Track ' WHERE track_id=22`, 3},
		{"artist case", "track", `UPDATE artist_aliases SET alias='source Artist' WHERE artist_id=30`, 3},
		{"album case", "album", `UPDATE release_aliases SET alias='source Album' WHERE release_id=21`, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := deletionFixture143(t)
			require.NoError(t, a.store.Exec(tc.change))
			id := 12
			if tc.kind == "album" {
				id = 11
			}
			base := fmt.Sprintf("/second/manage/%s/%d", tc.kind, id)
			v, w := deleteForm143(t, a, base)
			require.Contains(t, w.Body.String(), fmt.Sprintf(`data-delete-count="%d"`, tc.count))
			result := taskRequest139(a, "POST", base+"/confirm", a.cookie, v, nil)
			require.Equal(t, 303, result.Code, result.Body.String())
			var count int
			require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=22`).Scan(&count))
			require.Equal(t, 1, count)
		})
	}
	a := deletionFixture143(t)
	require.NoError(t, a.store.Exec(`UPDATE track_aliases SET alias='Source Track' WHERE track_id=52`))
	require.NoError(t, a.store.Exec(`INSERT INTO artist_tracks VALUES(10,52,0)`))
	v, w := deleteForm143(t, a, "/second/manage/track/42")
	require.Contains(t, w.Body.String(), `data-delete-count="2"`)
	result := taskRequest139(a, "POST", "/second/manage/track/42/confirm", a.cookie, v, nil)
	require.Equal(t, 303, result.Code, result.Body.String())
	var count int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=52`).Scan(&count))
	require.Equal(t, 1, count)
	// The same row count cannot conceal a changed set of reviewed artists.
	v, _ = deleteForm143(t, a, "/second/manage/track/12")
	require.NoError(t, a.store.Exec(`UPDATE artist_aliases SET alias='Renamed artist' WHERE artist_id=10`))
	result = taskRequest139(a, "POST", "/second/manage/track/12/confirm", a.cookie, v, nil)
	require.Equal(t, 409, result.Code)
	require.Equal(t, 404, taskRequest139(a, "GET", "/second/manage/track/999/delete", a.cookie, nil, nil).Code)
}

// TestProfileDeleteDerivedAlbumCredits143 uses the same live credits as the displayed album.
func TestProfileDeleteDerivedAlbumCredits143(t *testing.T) {
	a := deletionFixture143(t)
	// Both albums still credit A+B through their tracks after independent B credits are removed.
	require.NoError(t, a.store.Exec(`DELETE FROM artist_releases WHERE artist_id=20 AND release_id IN (41,61)`))
	v, w := deleteForm143(t, a, "/second/manage/album/41")
	require.Contains(t, w.Body.String(), `data-delete-count="2"`)
	post := taskRequest139(a, "POST", "/second/manage/album/41/confirm", a.cookie, v, nil)
	require.Equal(t, 303, post.Code, post.Body.String())
	require.Equal(t, "2", post.Header().Get("X-TuneTrail-Deleted-Listens"))
	var n int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id IN (42,62)`).Scan(&n))
	require.Zero(t, n)
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=12`).Scan(&n))
	require.Equal(t, 2, n)
}

// TestProfileDeleteSpentTicket143 isolates ticket replay from the already-consumed plan check.
func TestProfileDeleteSpentTicket143(t *testing.T) {
	a := deletionFixture143(t)
	first, _ := deleteForm143(t, a, "/second/manage/track/12")
	second, _ := deleteForm143(t, a, "/second/manage/track/32")
	post := taskRequest139(a, "POST", "/second/manage/track/12/confirm", a.cookie, first, nil)
	require.Equal(t, 303, post.Code, post.Body.String())
	second.Set("submission_token", first.Get("submission_token"))
	post = taskRequest139(a, "POST", "/second/manage/track/32/confirm", a.cookie, second, nil)
	require.Equal(t, 409, post.Code)
	var n int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=32`).Scan(&n))
	require.Equal(t, 1, n)
}

// TestProfileDeleteOtherProfileChange143 prevents unrelated listening updates from changing the reviewed scope.
func TestProfileDeleteOtherProfileChange143(t *testing.T) {
	a := deletionFixture143(t)
	v, _ := deleteForm143(t, a, "/second/manage/track/12")
	require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(3,12,999)`))
	post := taskRequest139(a, "POST", "/second/manage/track/12/confirm", a.cookie, v, nil)
	require.Equal(t, 303, post.Code, post.Body.String())
	require.Equal(t, "4", post.Header().Get("X-TuneTrail-Deleted-Listens"))
	var n int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=3 AND track_id=12`).Scan(&n))
	require.Equal(t, 2, n)
}
