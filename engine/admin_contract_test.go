package engine

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAdminMergeAPIContracts exercises real JSON routes without introducing HTML actions.
func TestAdminMergeAPIContracts(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		t.Run(kind, func(t *testing.T) {
			a := newProfileTestApp(t)
			for _, q := range []string{
				`INSERT INTO artists(id) VALUES(1),(2)`,
				`INSERT INTO artist_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
				`INSERT INTO releases(id,various_artists) VALUES(1,0),(2,0)`,
				`INSERT INTO release_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
				`INSERT INTO artist_releases VALUES(1,1,1),(2,2,1)`,
				`INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180),(2,2,120)`,
				`INSERT INTO track_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
				`INSERT INTO artist_tracks VALUES(1,1,1),(2,2,1)`,
				`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,100),(2,1,100),(1,2,100),(2,2,100)`,
			} {
				require.NoError(t, a.store.Exec(q))
			}
			base := "/apis/web/v1/" + kind
			for _, test := range []struct {
				to     string
				from   int
				status int
			}{{"1", 1, 400}, {"2", 0, 400}, {"2", -1, 400}, {"0", 1, 400}, {"2147483648", 1, 400}, {"2", 999, 404}, {"999", 1, 404}} {
				w := a.request("POST", base+"/"+test.to+"/merge", fmt.Sprintf(`{"merge_from_id":%d}`, test.from), "", true)
				require.Equal(t, test.status, w.Code, w.Body.String())
				require.NotContains(t, w.Body.String(), "SELECT")
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			}
			endpoint := base + "/2/merge"
			require.Equal(t, 401, a.request("POST", endpoint, `{"merge_from_id":1}`, "first-key", false).Code)
			req := httptest.NewRequest("POST", "http://koito.test"+endpoint, strings.NewReader(`{"merge_from_id":1}`))
			req.AddCookie(&http.Cookie{Name: "koito_session", Value: a.cookie})
			req.Header.Set("Origin", "https://other.invalid")
			w := httptest.NewRecorder()
			a.router.ServeHTTP(w, req)
			require.Equal(t, 403, w.Code)
			var before int
			require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens`).Scan(&before))
			require.Equal(t, 4, before)
			// Real SQL failures are logged internally, but the public response must be generic.
			require.NoError(t, a.store.Exec(`CREATE TRIGGER injected BEFORE UPDATE ON tracks BEGIN SELECT RAISE(ABORT,'private SQL detail'); END`))
			if kind == "album" {
				w = a.request("POST", endpoint, `{"merge_from_id":1}`, "", true)
				require.Equal(t, 500, w.Code)
				require.NotContains(t, w.Body.String(), "private SQL detail")
			}
			require.NoError(t, a.store.Exec(`DROP TRIGGER injected`))
			require.Equal(t, 204, a.request("POST", endpoint, `{"merge_from_id":1}`, "", true).Code)
			// A repeated merge cannot apply the same destructive change twice.
			require.Equal(t, 404, a.request("POST", endpoint, `{"merge_from_id":1}`, "", true).Code)
		})
	}
}

// TestAdminSessionCookiesAndPrivileges checks fixed expirations, no-store, and actor separation.
func TestAdminSessionCookiesAndPrivileges(t *testing.T) {
	a := newProfileTestApp(t)
	for _, remember := range []bool{false, true} {
		w := a.request("POST", "/apis/web/v1/login", fmt.Sprintf(`{"username":"administrator","password":"test-password","remember_me":%t}`, remember), "", false)
		require.Equal(t, 204, w.Code)
		cookies := w.Result().Cookies()
		require.Len(t, cookies, 1)
		cookie := cookies[0]
		require.True(t, cookie.HttpOnly)
		require.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		var expires int64
		require.NoError(t, a.store.QueryRow(`SELECT expires_at FROM sessions WHERE id=?`, cookie.Value).Scan(&expires))
		require.Equal(t, expires, cookie.Expires.Unix())
		for i := 0; i < 2; i++ {
			r := httptest.NewRequest("GET", "http://koito.test/apis/web/v1/user", nil)
			r.AddCookie(cookie)
			w = httptest.NewRecorder()
			a.router.ServeHTTP(w, r)
			require.Equal(t, 200, w.Code)
		}
		var after int64
		require.NoError(t, a.store.QueryRow(`SELECT expires_at FROM sessions WHERE id=?`, cookie.Value).Scan(&after))
		require.Equal(t, expires, after)
	}
	user, e := a.store.SaveUser(context.Background(), db.SaveUserOpts{Username: "not-admin", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, e)
	session, e := a.store.SaveSession(context.Background(), user.ID, time.Now().Add(time.Hour), false)
	require.NoError(t, e)
	r := httptest.NewRequest("GET", "http://koito.test/apis/web/v1/user", nil)
	r.AddCookie(&http.Cookie{Name: "koito_session", Value: session.ID.String()})
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
	require.NoError(t, a.store.Exec(`UPDATE sessions SET expires_at=1 WHERE id=?`, a.cookie))
	require.Equal(t, 401, a.request("GET", "/apis/web/v1/user", "", "", true).Code)
}
