package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/stretchr/testify/require"
)

func TestCatalogReviewStaleAndOneUse(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	for _, q := range []string{`INSERT INTO artists(id) VALUES(10)`, `INSERT INTO artist_aliases VALUES(10,'Fictional Artist','User',1)`, `INSERT INTO releases(id,various_artists) VALUES(11,0)`, `INSERT INTO release_aliases VALUES(11,'Fictional Album','User',1)`, `INSERT INTO artist_releases VALUES(10,11,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(12,11,60),(13,11,60)`, `INSERT INTO track_aliases VALUES(12,'Source','User',1),(13,'Destination','User',1)`, `INSERT INTO artist_tracks VALUES(10,12,1),(10,13,1)`, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,100),(3,12,100),(2,13,100)`} {
		require.NoError(t, a.store.Exec(q))
	}
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	call := func(method, path string, values url.Values, auth bool) *httptest.ResponseRecorder {
		body := ""
		if values != nil {
			body = values.Encode()
		}
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if values != nil {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if auth {
			r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	base := "/second/manage/track/12"
	review := func() *httptest.ResponseRecorder { return call("GET", base+"/review?action=merge&to=13", nil, true) }
	extract := func(w *httptest.ResponseRecorder, name string) string {
		t.Helper()
		m := regexp.MustCompile(`name="` + name + `" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
		require.Len(t, m, 2, w.Body.String())
		return m[1]
	}
	w := review()
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "Destination (ID 13)")
	require.Contains(t, w.Body.String(), "Duplicate listens in the same profile and second: 1")
	v := url.Values{"csrf_token": {extract(w, "csrf_token")}, "profile_id": {"2"}, "profile_slug": {"second"}, "plan": {extract(w, "plan")}, "confirm": {"1"}}
	user, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	secondSession, err := a.store.SaveSession(ctx, user.ID, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	foreign := httptest.NewRequest("POST", base+"/confirm", strings.NewReader(v.Encode()))
	foreign.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	foreign.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: secondSession.ID.String()})
	foreignW := httptest.NewRecorder()
	h.ServeHTTP(foreignW, foreign)
	require.Equal(t, 403, foreignW.Code)
	wrongRoute := call("POST", "/second/manage/track/13/confirm", v, true)
	require.Equal(t, 409, wrongRoute.Code)
	inverse := call("GET", base+"/review?action=merge&to=13&reverse=1", nil, true)
	require.Equal(t, 200, inverse.Code)
	require.Contains(t, inverse.Body.String(), "Source: Destination (ID 13)")
	require.Contains(t, inverse.Body.String(), "Source (ID 12)")
	visitor := call("GET", base+"/review?action=delete", nil, false)
	require.Equal(t, 303, visitor.Code)
	require.NoError(t, a.store.Exec(`UPDATE track_aliases SET source='changed' WHERE track_id=12`))
	w = call("POST", base+"/confirm", v, true)
	require.Equal(t, 409, w.Code)
	memkv.Store.Set("now-playing:2", int32(12))
	memkv.Store.Set("now-playing:3", int32(13))
	w = review()
	require.Equal(t, 200, w.Code)
	v.Set("plan", extract(w, "plan"))
	v.Set("csrf_token", "bad")
	require.Equal(t, 403, call("POST", base+"/confirm", v, true).Code)
	v.Set("csrf_token", extract(w, "csrf_token"))
	w = call("POST", base+"/confirm", v, true)
	require.Equal(t, 303, w.Code, w.Body.String())
	require.Equal(t, "/second", w.Header().Get("Location"))
	_, playing := memkv.Store.Get("now-playing:2")
	require.False(t, playing)
	_, playing = memkv.Store.Get("now-playing:3")
	require.False(t, playing)
	require.Equal(t, 409, call("POST", base+"/confirm", v, true).Code)
	_, err = a.store.GetTrack(ctx, db.GetTrackOpts{ID: 12})
	require.ErrorIs(t, err, db.ErrNotFound)
	n, err := a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	n, err = a.store.ProfileListenCount(ctx, 3)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}
func TestProfileReviewAndStaleCount(t *testing.T) {
	a := newProfileTestApp(t)
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	r := httptest.NewRequest("GET", "/admin/profile/2/delete-review", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "shared catalog remain")
	re := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	m := re.FindStringSubmatch(w.Body.String())
	require.Len(t, m, 2)
	v := url.Values{"csrf_token": {m[1]}, "profile_slug": {"second"}, "display_name": {"Second"}, "listen_count": {"1"}, "confirm": {"1"}}
	r = httptest.NewRequest("POST", "/admin/profile/2/delete", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 409, w.Code)
	v.Set("listen_count", "0")
	r = httptest.NewRequest("POST", "/admin/profile/2/delete", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 303, w.Code)
	_, err = a.store.GetListeningProfileByID(context.Background(), 2)
	require.ErrorIs(t, err, db.ErrNotFound)
}

func TestDeleteLastProfileKeepsAdministrator(t *testing.T) {
	a := newProfileTestApp(t)
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	for _, p := range []struct {
		id         int
		slug, name string
	}{{1, "default", "administrator"}, {2, "second", "Second"}, {3, "empty", "Empty"}} {
		r := httptest.NewRequest("GET", fmt.Sprintf("/admin/profile/%d/delete-review", p.id), nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, 200, w.Code)
		match := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
		require.Len(t, match, 2)
		v := url.Values{"csrf_token": {match[1]}, "profile_slug": {p.slug}, "display_name": {p.name}, "listen_count": {"0"}, "confirm": {"1"}}
		r = httptest.NewRequest("POST", fmt.Sprintf("/admin/profile/%d/delete", p.id), strings.NewReader(v.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, 303, w.Code, w.Body.String())
	}
	profiles, err := a.store.ListListeningProfiles(context.Background())
	require.NoError(t, err)
	require.Empty(t, profiles)
	r := httptest.NewRequest("GET", "/admin", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "No profiles yet")
}
