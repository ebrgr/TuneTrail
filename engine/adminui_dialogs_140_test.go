package engine

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestMediaTaskDialog140 verifies initial tickets, scoped task content and rotation through the real router.
func TestMediaTaskDialog140(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"}
	for _, task := range []string{"edit", "image", "add-listen", "merge", "delete"} {
		w := taskRequest139(a, "GET", base+"/"+task+"?partial=1", a.cookie, nil, headers)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.Equal(t, "task", w.Header().Get("X-TuneTrail-Fragment"))
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		// Personal deletion now issues its required submission ticket immediately (ISSUE-143).
		if task == "merge" {
			require.Empty(t, w.Header().Get("X-TuneTrail-Submission"))
		} else {
			require.NotEmpty(t, w.Header().Get("X-TuneTrail-Submission"))
		}
		require.NotContains(t, w.Body.String(), "<html")
		require.NotContains(t, w.Body.String(), "<h1")
		if task == "add-listen" {
			require.NotContains(t, w.Body.String(), "listen-delete")
		}
		if task != "image" {
			require.NotContains(t, w.Body.String(), "/artwork\"")
		}
	}
	initial := taskRequest139(a, "GET", base+"/edit?partial=1", a.cookie, nil, headers)
	ticket := taskField139(t, initial, "submission_token")
	headers["X-TuneTrail-Submission"] = ticket
	search := taskRequest139(a, "GET", base+"/edit?partial=1&artist_query=Source", a.cookie, nil, headers)
	require.Equal(t, 200, search.Code, search.Body.String())
	require.Equal(t, ticket, search.Header().Get("X-TuneTrail-Submission"))
	form := url.Values{"csrf_token": {taskField139(t, initial, "csrf_token")}, "submission_token": {ticket}, "return_to": {base + "/edit"}, "profile_id": {"2"}, "profile_slug": {"second"}, "alias": {"Dialog alias"}}
	post := taskRequest139(a, "POST", base+"/alias-add", a.cookie, form, headers)
	require.Equal(t, 200, post.Code, post.Body.String())
	require.Equal(t, "task", post.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, post.Body.String(), "Dialog alias")
	require.NotEqual(t, ticket, post.Header().Get("X-TuneTrail-Submission"))
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/alias-add", a.cookie, form, headers).Code)
	// A failed mutation also returns a fresh usable ticket and keeps the exact task.
	form.Set("submission_token", post.Header().Get("X-TuneTrail-Submission"))
	form.Set("alias", "")
	failed := taskRequest139(a, "POST", base+"/alias-add", a.cookie, form, headers)
	require.Equal(t, 400, failed.Code, failed.Body.String())
	require.Equal(t, "task", failed.Header().Get("X-TuneTrail-Fragment"))
	require.NotEmpty(t, failed.Header().Get("X-TuneTrail-Submission"))
	require.Contains(t, failed.Body.String(), "/alias-add")
}

// TestMediaDialogSecurity140 keeps authentication, CSRF, profile scope and destructive plans authoritative.
func TestMediaDialogSecurity140(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"}
	for _, cookie := range []string{"", "expired"} {
		w := taskRequest139(a, "GET", base+"/edit?partial=1", cookie, nil, headers)
		require.Empty(t, w.Header().Get("X-TuneTrail-Fragment"))
		require.NotContains(t, w.Body.String(), "submission_token")
	}
	navigation := taskRequest139(a, "GET", base+"/edit?partial=1", a.cookie, nil, nil)
	require.Equal(t, 303, navigation.Code)
	require.Equal(t, base+"/edit", navigation.Header().Get("Location"))
	foreign := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1", "Origin": "https://foreign.test", "Sec-Fetch-Site": "cross-site"}
	require.Equal(t, 403, taskRequest139(a, "GET", base+"/edit?partial=1", a.cookie, nil, foreign).Code)
	for _, violation := range []string{"csrf", "profile", "ticket"} {
		form := taskForm139(t, a, base+"/edit")
		form.Set("alias", "Forbidden")
		switch violation {
		case "csrf":
			form.Set("csrf_token", "bad")
		case "profile":
			form.Set("profile_id", "3")
		case "ticket":
			form.Set("submission_token", "")
		}
		w := taskRequest139(a, "POST", base+"/alias-add", a.cookie, form, headers)
		require.Contains(t, []int{403, 409}, w.Code, violation+": "+w.Body.String())
	}
	review := taskRequest139(a, "GET", base+"/review?action=merge&to=13&partial=1", a.cookie, nil, headers)
	require.Equal(t, 200, review.Code, review.Body.String())
	require.Equal(t, "review", review.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, "no-store", review.Header().Get("Cache-Control"))
	require.NotContains(t, review.Body.String(), "<h1")
	require.NotContains(t, review.Body.String(), "<html")
	form := url.Values{"csrf_token": {taskField139(t, review, "csrf_token")}, "plan": {taskField139(t, review, "plan")}, "profile_id": {"2"}, "profile_slug": {"second"}, "confirm": {"1"}}
	confirmed := taskRequest139(a, "POST", base+"/confirm", a.cookie, form, headers)
	require.Equal(t, 204, confirmed.Code, confirmed.Body.String())
	require.Equal(t, "/second/manage/track/13", confirmed.Header().Get("X-TuneTrail-Navigate"))
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/confirm", a.cookie, form, headers).Code)
	// A real expired server-side session must not restore forms or administrator controls.
	require.NoError(t, a.store.Exec(`UPDATE sessions SET expires_at=1 WHERE id=?`, a.cookie))
	expired := taskRequest139(a, "GET", base+"/edit?partial=1", a.cookie, nil, headers)
	require.NotEqual(t, 200, expired.Code)
	require.Empty(t, expired.Header().Get("X-TuneTrail-Fragment"))
	require.NotContains(t, expired.Body.String(), "submission_token")
	require.Equal(t, 401, taskRequest139(a, "POST", base+"/confirm", a.cookie, form, headers).Code)
	public := taskRequest139(a, "GET", "/second/track/12", a.cookie, nil, nil)
	require.Equal(t, 200, public.Code)
	require.NotContains(t, public.Body.String(), "data-media-task")
	require.Equal(t, "false", public.Header().Get("X-TuneTrail-Admin"))
}

// TestMediaActionVisibility140 verifies server-side role decisions and the real no-JavaScript hrefs.
func TestMediaActionVisibility140(t *testing.T) {
	a := taskApp139(t)
	for _, item := range []struct{ kind, id string }{{"artist", "10"}, {"album", "11"}, {"track", "12"}} {
		path := "/second/" + item.kind + "/" + item.id
		visitor := taskRequest139(a, "GET", path, "", nil, nil)
		require.Equal(t, 200, visitor.Code, visitor.Body.String())
		require.NotContains(t, visitor.Body.String(), "data-media-task")
		require.NotContains(t, visitor.Body.String(), "media-task-dialog")
		require.NotContains(t, visitor.Body.String(), "media-dialog.js")
		require.NotContains(t, visitor.Body.String(), "media-dialog.css")
		require.NotContains(t, visitor.Body.String(), "Manage this item")
		admin := taskRequest139(a, "GET", path, a.cookie, nil, nil)
		require.Equal(t, 200, admin.Code, admin.Body.String())
		require.Contains(t, admin.Body.String(), "media-dialog.js")
		base := "/second/manage/" + item.kind + "/" + item.id
		edit := base + "/edit"
		require.Contains(t, admin.Body.String(), `data-media-task href="`+edit+`"`)
		// MusicBrainz is public and precedes management actions when metadata is present.
		form := taskForm139(t, a, edit)
		form.Set("mbid", "b24d0e8e-07dd-4028-8e65-7f2168ea9a22")
		if item.kind == "album" {
			form.Set("various", "0")
		}
		require.Equal(t, 303, taskRequest139(a, "POST", base+"/metadata", a.cookie, form, nil).Code)
		withMBID := taskRequest139(a, "GET", path, a.cookie, nil, nil).Body.String()
		require.Less(t, strings.Index(withMBID, "https://musicbrainz.org/"), strings.Index(withMBID, "data-media-task"))
		visitorMBID := taskRequest139(a, "GET", path, "", nil, nil).Body.String()
		require.Contains(t, visitorMBID, "https://musicbrainz.org/")
		require.NotContains(t, visitorMBID, "data-media-task")
	}
}

// TestMediaOptionalTimezone140 preserves old POST fields and resolves a blank zone on the server.
func TestMediaOptionalTimezone140(t *testing.T) {
	for _, tc := range []struct {
		name, zone, pageZone, wall, instant string
		status                              int
	}{
		{"blank UTC", "", "UTC", "2023-02-03T12:34", "2023-02-03T12:34:00Z", 200},
		{"blank return zone", "", "Asia/Tokyo", "2023-02-03T12:34", "2023-02-03T03:34:00Z", 200},
		{"explicit old contract", "America/Sao_Paulo", "UTC", "2023-02-03T12:34", "2023-02-03T15:34:00Z", 200},
		{"invalid zone", "No/Such_Zone", "UTC", "2023-02-03T12:34", "", 400},
		{"nonexistent wall time", "America/Sao_Paulo", "UTC", "2018-11-04T00:30", "", 400},
		{"ambiguous wall time", "America/Sao_Paulo", "UTC", "2019-02-16T23:30", "", 409},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := taskApp139(t)
			base := "/second/manage/track/12"
			form := taskForm139(t, a, base+"/add-listen?tz="+url.QueryEscape(tc.pageZone))
			form.Set("local_time", tc.wall)
			form.Set("timezone", tc.zone)
			form.Set("offset", "")
			headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"}
			result := taskRequest139(a, "POST", base+"/listen-add", a.cookie, form, headers)
			require.Equal(t, tc.status, result.Code, result.Body.String())
			var count int
			require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=12`).Scan(&count))
			if tc.status == 200 {
				require.Equal(t, 2, count)
				expected, err := time.Parse(time.RFC3339, tc.instant)
				require.NoError(t, err)
				require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=12 AND listened_at=?`, expected.Unix()).Scan(&count))
				require.Equal(t, 1, count)
			} else {
				require.Equal(t, 1, count)
				require.NotEmpty(t, result.Header().Get("X-TuneTrail-Submission"))
				if tc.status == 409 {
					require.Contains(t, result.Body.String(), `class="listen-offset" open`)
				}
				if tc.zone == "No/Such_Zone" {
					require.Contains(t, result.Body.String(), "Invalid time zone.")
					require.Contains(t, result.Body.String(), tc.zone)
				}
			}
		})
	}
}

// TestMediaLiveSearch140 returns only selectable results, without issuing mutation tickets per query.
func TestMediaLiveSearch140(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1", "X-TuneTrail-Search": "merge"}
	result := taskRequest139(a, "GET", base+"/merge?merge_query=Target&partial=1", a.cookie, nil, headers)
	require.Equal(t, 200, result.Code, result.Body.String())
	require.Equal(t, "merge-results", result.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, "no-store", result.Header().Get("Cache-Control"))
	require.Empty(t, result.Header().Get("X-TuneTrail-Submission"))
	require.Contains(t, result.Body.String(), "Target Track")
	require.Contains(t, result.Body.String(), "/review?action=merge&amp;to=13")
	require.NotContains(t, result.Body.String(), "merge_query")
	edit := taskForm139(t, a, base+"/edit")
	headers["X-TuneTrail-Search"] = "artists"
	headers["X-TuneTrail-Submission"] = edit.Get("submission_token")
	result = taskRequest139(a, "GET", base+"/edit?artist_query=Target&partial=1", a.cookie, nil, headers)
	require.Equal(t, 200, result.Code, result.Body.String())
	require.Equal(t, "artist-results", result.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, edit.Get("submission_token"), result.Header().Get("X-TuneTrail-Submission"))
	require.Contains(t, result.Body.String(), "Target Artist")
	require.NotContains(t, result.Body.String(), "<html")
}

// TestMediaInvalidReturnTimezone140 proves invalid return context cannot be opened or used to insert a listen.
func TestMediaInvalidReturnTimezone140(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	require.Equal(t, 400, taskRequest139(a, "GET", base+"/add-listen?tz=Invalid/Zone", a.cookie, nil, nil).Code)
	form := taskForm139(t, a, base+"/add-listen?tz=UTC")
	form.Set("return_to", base+"/add-listen?tz=Invalid/Zone")
	form.Set("local_time", "2023-02-03T12:34")
	form.Set("timezone", "")
	form.Set("offset", "")
	result := taskRequest139(a, "POST", base+"/listen-add", a.cookie, form, map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"})
	require.Equal(t, 400, result.Code)
	var count int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM listens WHERE profile_id=2 AND track_id=12`).Scan(&count))
	require.Equal(t, 1, count)
}
