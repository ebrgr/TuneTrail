package engine

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestListenDialog141 shares page content and one-use authorization with the native dialog.
func TestListenDialog141(t *testing.T) {
	a := taskApp139(t)
	require.NoError(t, a.store.Exec(`UPDATE listens SET client='Player <one>' WHERE profile_id=2`))
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"}
	base := "/second/manage/track/12"
	back := "/second/track/12?tz=Asia%2FTokyo"
	page := base + "/listen?listen_at=100&tz=Asia%2FTokyo&history_return=" + url.QueryEscape(back)
	full := taskRequest139(a, "GET", page, a.cookie, nil, nil)
	partial := taskRequest139(a, "GET", page+"&partial=1", a.cookie, nil, headers)
	require.Equal(t, 200, partial.Code, partial.Body.String())
	require.Equal(t, "task", partial.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, "no-store", partial.Header().Get("Cache-Control"))
	for _, text := range []string{"Source Track", "Source Artist", "Source Album", "Player &lt;one&gt;", "1970-01-01 09:01:40 JST", "This cannot be undone.", `/second/track/12`, `/second/artist/10`, `/second/album/11`} {
		require.Contains(t, full.Body.String(), text)
		require.Contains(t, partial.Body.String(), text)
	}
	require.NotContains(t, partial.Body.String(), "Catalog changes affect all profiles")
	form := url.Values{"csrf_token": {taskField139(t, partial, "csrf_token")}, "submission_token": {taskField139(t, partial, "submission_token")}, "profile_id": {"2"}, "profile_slug": {"second"}, "return_to": {back}, "unix": {"100"}}
	missing := taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, headers)
	require.Equal(t, 400, missing.Code, missing.Body.String())
	require.Equal(t, "task", missing.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, missing.Body.String(), "Player &lt;one&gt;")
	require.Contains(t, missing.Body.String(), "1970-01-01 09:01:40 JST")
	require.Contains(t, missing.Body.String(), "selected-listen")
	require.NotEqual(t, form.Get("submission_token"), taskField139(t, missing, "submission_token"))
	form.Set("submission_token", taskField139(t, missing, "submission_token"))
	form.Set("confirm", "1")
	done := taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, headers)
	require.Equal(t, 204, done.Code, done.Body.String())
	require.Equal(t, "100", done.Header().Get("X-TuneTrail-Listen-Deleted"))
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, headers).Code)
	require.Equal(t, 404, taskRequest139(a, "GET", page, a.cookie, nil, nil).Code)
	require.Equal(t, 200, taskRequest139(a, "GET", strings.ReplaceAll(page, "second", "empty"), a.cookie, nil, nil).Code)
	for _, route := range []string{"track/12", "album/11", "artist/10"} {
		// ISSUE-150: a plain administrator GET (reload) shows the empty state like the dialog refresh.
		reload := taskRequest139(a, "GET", "/second/"+route, a.cookie, nil, nil)
		require.Equal(t, 200, reload.Code, reload.Body.String())
		require.Contains(t, reload.Body.String(), `data-listen-count="0"`)
		require.Contains(t, reload.Body.String(), "Nothing to show on this page.")
		require.Equal(t, 404, taskRequest139(a, "GET", "/second/"+route, "", nil, nil).Code)
		// A missing entity stays 404 even for the administrator.
		require.Equal(t, 404, taskRequest139(a, "GET", "/second/"+strings.Split(route, "/")[0]+"/999999", a.cookie, nil, nil).Code)
		snapshot := taskRequest139(a, "GET", "/second/"+route, a.cookie, nil, map[string]string{"X-TuneTrail-History-Refresh": "1"})
		require.Equal(t, 200, snapshot.Code, snapshot.Body.String())
		require.Contains(t, snapshot.Body.String(), `data-listen-count="0"`)
		require.Contains(t, snapshot.Body.String(), "Nothing to show on this page.")
		require.Equal(t, 404, taskRequest139(a, "GET", "/second/"+route, "", nil, map[string]string{"X-TuneTrail-History-Refresh": "1"}).Code)
	}
}

// TestListenSecurity141 rejects forged context before deletion through the production router.
func TestListenSecurity141(t *testing.T) {
	for _, violation := range []string{"csrf", "ticket", "profile", "slug", "cross-profile-return", "origin", "session"} {
		t.Run(violation, func(t *testing.T) {
			a := taskApp139(t)
			base := "/second/manage/track/12"
			form := taskForm139(t, a, base+"/listen?listen_at=100&tz=UTC")
			form.Set("unix", "100")
			form.Set("confirm", "1")
			cookie := a.cookie
			headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"}
			switch violation {
			case "csrf":
				form.Set("csrf_token", "bad")
			case "ticket":
				form.Set("submission_token", "")
			case "profile":
				form.Set("profile_id", "3")
			case "slug":
				form.Set("profile_slug", "empty")
			case "cross-profile-return":
				form.Set("return_to", "/empty/track/12?tz=UTC")
			case "origin":
				headers["Origin"] = "https://foreign.test"
			case "session":
				cookie = ""
			}
			out := taskRequest139(a, "POST", base+"/listen-delete", cookie, form, headers)
			require.Contains(t, []int{400, 401, 403, 409}, out.Code, out.Body.String())
			require.Equal(t, 200, taskRequest139(a, "GET", base+"/listen?listen_at=100", a.cookie, nil, nil).Code)
		})
	}
	a := taskApp139(t)
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1", "Origin": "https://foreign.test"}
	path := "/second/manage/track/12/listen?listen_at=100&partial=1"
	require.Equal(t, 403, taskRequest139(a, "GET", path, a.cookie, nil, headers).Code)
	for _, cookie := range []string{"", "expired"} {
		out := taskRequest139(a, "GET", path, cookie, nil, map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"})
		require.Empty(t, out.Header().Get("X-TuneTrail-Fragment"))
		require.NotContains(t, out.Body.String(), "submission_token")
	}
}

// TestListenHistoryControls141 keeps management out of visitor HTML and returns to real media pages.
func TestListenHistoryControls141(t *testing.T) {
	a := taskApp139(t)
	for _, route := range []string{"", "/listens", "/track/12", "/album/11", "/artist/10"} {
		path := "/second" + route + "?tz=UTC"
		visitor := taskRequest139(a, "GET", path, "", nil, nil)
		require.Equal(t, 200, visitor.Code)
		require.NotContains(t, visitor.Body.String(), "listen-manage")
		require.NotContains(t, visitor.Body.String(), "/manage/track/12/listen?")
		require.NotContains(t, visitor.Body.String(), "media-dialog.js")
		admin := taskRequest139(a, "GET", path, a.cookie, nil, nil)
		require.Equal(t, 200, admin.Code)
		require.Contains(t, admin.Body.String(), "listen-manage")
		require.Contains(t, admin.Body.String(), "/manage/track/12/listen?listen_at=100")
		require.Contains(t, admin.Body.String(), "media-dialog.js")
		require.Contains(t, admin.Body.String(), `id="media-task-dialog"`)
		require.NotContains(t, admin.Body.String(), "admin.js")
		if strings.Contains(route, "/track/") || strings.Contains(route, "/album/") || strings.Contains(route, "/artist/") {
			require.Contains(t, admin.Body.String(), "history_return="+url.QueryEscape(path))
		}
	}
}

// TestListenMediaReturn141 retains each originating media page for ordinary form navigation.
func TestListenMediaReturn141(t *testing.T) {
	for _, route := range []string{"track/12", "album/11", "artist/10"} {
		t.Run(route, func(t *testing.T) {
			a := taskApp139(t)
			require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,101)`))
			back := "/second/" + route + "?tz=UTC"
			base := "/second/manage/track/12"
			form := taskForm139(t, a, base+"/listen?listen_at=100&tz=UTC&history_return="+url.QueryEscape(back))
			require.Equal(t, back, form.Get("return_to"))
			form.Set("unix", "100")
			form.Set("confirm", "1")
			out := taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, nil)
			require.Equal(t, 303, out.Code, out.Body.String())
			require.Equal(t, back, out.Header().Get("Location"))
			require.Equal(t, 200, taskRequest139(a, "GET", back, a.cookie, nil, nil).Code)
			require.Equal(t, 400, taskRequest139(a, "GET", base+"/listen?listen_at=101&history_return="+url.QueryEscape("/empty/"+route), a.cookie, nil, nil).Code)
		})
	}
}
