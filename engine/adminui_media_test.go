package engine

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/stretchr/testify/require"
)

func TestPreparedMediaFormsAndProfileContext(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	for _, q := range []string{`INSERT INTO artists(id) VALUES(10)`, `INSERT INTO artist_aliases VALUES(10,'Fictional Artist','User',1)`, `INSERT INTO releases(id,various_artists) VALUES(11,0)`, `INSERT INTO release_aliases VALUES(11,'Fictional Album','User',1)`, `INSERT INTO artist_releases VALUES(10,11,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(12,11,60)`, `INSERT INTO track_aliases VALUES(12,'Fictional Track','User',1)`, `INSERT INTO artist_tracks VALUES(10,12,1)`} {
		require.NoError(t, a.store.Exec(q))
	}
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	call := func(method, path string, values url.Values, auth bool) *httptest.ResponseRecorder {
		var body *strings.Reader
		if values != nil {
			body = strings.NewReader(values.Encode())
		} else {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		if values != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if auth {
			req.AddCookie(&http.Cookie{Name: "koito_session", Value: a.cookie})
		}
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		return out
	}
	base := "/second/manage/track/12"
	page := call("GET", base, nil, true)
	require.Equal(t, 200, page.Code, page.Body.String())
	require.Contains(t, page.Body.String(), "album ID 11")
	require.Contains(t, page.Header().Get("Cache-Control"), "no-store")
	match := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(page.Body.String())
	require.Len(t, match, 2)
	v := url.Values{"csrf_token": {match[1]}, "profile_id": {"2"}, "profile_slug": {"second"}, "alias": {"Ficção 音"}}
	require.Equal(t, 303, call("POST", base+"/alias-add", v, true).Code)
	aliases, err := a.store.GetAllTrackAliases(ctx, 12)
	require.NoError(t, err)
	require.Len(t, aliases, 2)
	v.Set("profile_id", "3")
	require.Equal(t, 409, call("POST", base+"/alias-add", v, true).Code)
	v.Set("profile_id", "2")
	v.Set("csrf_token", "wrong")
	require.Equal(t, 403, call("POST", base+"/alias-add", v, true).Code)
	redirect := call("GET", base+"?artist_query=Fictional", nil, false)
	require.Equal(t, 303, redirect.Code)
	require.Contains(t, redirect.Header().Get("Location"), "next=")
	require.Contains(t, redirect.Header().Get("Location"), "artist_query")
	invalid := url.Values{"csrf_token": {match[1]}, "profile_id": {"2"}, "profile_slug": {"second"}, "mbid": {"not-a-uuid"}}
	invalid.Set("return_to", base+"?artist_query=Fake&tz=America%2FSao_Paulo")
	bad := call("POST", base+"/metadata", invalid, true)
	require.Equal(t, 400, bad.Code)
	require.Contains(t, bad.Body.String(), `value="not-a-uuid"`)
	require.Contains(t, bad.Body.String(), `value="America/Sao_Paulo"`)
	require.Contains(t, bad.Body.String(), `value="Fake"`)
	ambiguous := url.Values{"csrf_token": {match[1]}, "profile_id": {"2"}, "profile_slug": {"second"}, "local_time": {"2024-11-03T01:30"}, "timezone": {"America/New_York"}}
	bad = call("POST", base+"/listen-add", ambiguous, true)
	require.Equal(t, 409, bad.Code)
	require.Contains(t, bad.Body.String(), "occurs twice")
	require.Contains(t, bad.Body.String(), `value="America/New_York"`)
	v = url.Values{"csrf_token": {match[1]}, "profile_id": {"2"}, "profile_slug": {"second"}, "local_time": {"2023-11-14T19:13"}, "timezone": {"America/Sao_Paulo"}}
	out := call("POST", base+"/listen-add", v, true)
	require.Equal(t, 303, out.Code, fmt.Sprint(out.Body))
	count, err := a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Equal(t, 303, call("POST", base+"/listen-add", v, true).Code)
	count, err = a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.NoError(t, a.store.SaveListen(ctx, db.SaveListenOpts{ProfileID: 3, TrackID: 12, Time: time.Unix(1699999980, 0)}))
	deletion := url.Values{"csrf_token": {match[1]}, "profile_id": {"2"}, "profile_slug": {"second"}, "unix": {"1699999980"}}
	require.Equal(t, 400, call("POST", base+"/listen-delete", deletion, true).Code)
	deletion.Set("confirm", "1")
	require.Equal(t, 303, call("POST", base+"/listen-delete", deletion, true).Code)
	count, err = a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = a.store.ProfileListenCount(ctx, 3)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
