package engine

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// taskApp139 exercises the production router, whose administrative handler requires tickets.
func taskApp139(t *testing.T) *profileTestApp {
	t.Helper()
	a := newProfileTestApp(t)
	for _, sql := range []string{
		`INSERT INTO artists(id) VALUES(10),(20)`,
		`INSERT INTO artist_aliases VALUES(10,'Source Artist','User',1),(20,'Target Artist','User',1)`,
		`INSERT INTO releases(id,various_artists) VALUES(11,0)`,
		`INSERT INTO release_aliases VALUES(11,'Source Album','User',1)`,
		`INSERT INTO artist_releases VALUES(10,11,1)`,
		`INSERT INTO tracks(id,release_id,duration) VALUES(12,11,60),(13,11,60)`,
		`INSERT INTO track_aliases VALUES(12,'Source Track','User',1),(13,'Target Track','User',1)`,
		`INSERT INTO artist_tracks VALUES(10,12,1),(10,13,1)`,
		`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,100),(3,12,100)`,
	} {
		require.NoError(t, a.store.Exec(sql))
	}
	return a
}

// taskRequest139 keeps headers and credentials explicit for security boundary cases.
func taskRequest139(a *profileTestApp, method, path, cookie string, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	body := ""
	if form != nil {
		body = form.Encode()
	}
	// Browsers never send the client-side fragment in the HTTP request target.
	requestPath := strings.SplitN(path, "#", 2)[0]
	r := httptest.NewRequest(method, "http://koito.test"+requestPath, strings.NewReader(body))
	if form != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: cookie})
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	return w
}

// taskField139 reads real hidden inputs, including escaped return query values.
func taskField139(t *testing.T, w *httptest.ResponseRecorder, name string) string {
	t.Helper()
	match := regexp.MustCompile(`name="` + name + `"\s+value="([^"]*)"`).FindStringSubmatch(w.Body.String())
	require.Len(t, match, 2, name+": "+w.Body.String())
	return html.UnescapeString(match[1])
}

// taskForm139 submits the tokens emitted by the actual task document.
func taskForm139(t *testing.T, a *profileTestApp, path string) url.Values {
	t.Helper()
	w := taskRequest139(a, "GET", path, a.cookie, nil, nil)
	require.Equal(t, 200, w.Code, w.Body.String())
	// Ignore the language picker: submit the actual administrative form's return target.
	for _, body := range regexp.MustCompile(`(?s)<form\b[^>]*>.*?</form>`).FindAllString(w.Body.String(), -1) {
		if strings.Contains(body, `name="csrf_token"`) && strings.Contains(body, `name="submission_token"`) {
			formPage := httptest.NewRecorder()
			formPage.Body.WriteString(body)
			w = formPage
			break
		}
	}
	return url.Values{
		"csrf_token":       {taskField139(t, w, "csrf_token")},
		"submission_token": {taskField139(t, w, "submission_token")},
		"profile_id":       {"2"}, "profile_slug": {"second"},
		"return_to": {taskField139(t, w, "return_to")},
	}
}

// TestManagementTaskPages139 checks task separation, type restrictions and safe legacy navigation.
func TestManagementTaskPages139(t *testing.T) {
	a := taskApp139(t)
	for _, media := range []struct{ kind, id string }{{"track", "12"}, {"album", "11"}, {"artist", "10"}} {
		base := "/second/manage/" + media.kind + "/" + media.id
		index := taskRequest139(a, "GET", base, a.cookie, nil, nil)
		require.Equal(t, 200, index.Code)
		require.NotContains(t, index.Body.String(), `name="csrf_token"`)
		for _, anchor := range []string{"aliases", "metadata", "artists", "merge", "artwork", "delete"} {
			require.Contains(t, index.Body.String(), `id="`+anchor+`"`)
		}
		for _, task := range []string{"edit", "image", "merge", "delete", "add-listen", "listens", "listen"} {
			path := base + "/" + task
			if task == "listen" {
				path += "?listen_at=100&tz=UTC"
			}
			w := taskRequest139(a, "GET", path, a.cookie, nil, nil)
			if media.kind != "track" && (task == "add-listen" || task == "listens" || task == "listen") {
				require.Equal(t, 404, w.Code, path)
				require.NotContains(t, index.Body.String(), base+"/"+task)
				continue
			}
			require.Equal(t, 200, w.Code, path+": "+w.Body.String())
			require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
			if task != "merge" && task != "delete" {
				require.Contains(t, w.Body.String(), `name="csrf_token"`)
				require.NotEmpty(t, taskField139(t, w, "submission_token"))
			}
			body := w.Body.String()
			switch task {
			case "edit":
				require.Contains(t, body, `/alias-add"`)
				require.Contains(t, body, `/metadata"`)
				require.NotContains(t, body, `/listen-add"`)
				require.NotContains(t, body, `/artwork"`)
				require.NotContains(t, body, `/review"`)
			case "image":
				require.Contains(t, body, `/artwork"`)
				require.NotContains(t, body, `/alias-add"`)
				require.NotContains(t, body, `/review"`)
				if media.kind == "track" {
					require.Contains(t, body, "album ID 11")
				}
			case "merge":
				require.Contains(t, body, `value="merge"`)
				require.NotContains(t, body, `value="delete"`)
			case "delete":
				// Deletion now confirms personal listens directly (ISSUE-143).
				require.Contains(t, body, `data-delete-count=`)
				require.NotContains(t, body, `name="merge_query"`)
			case "add-listen":
				require.Contains(t, body, `/listen-add"`)
				require.NotContains(t, body, `/listen-delete"`)
			case "listens", "listen":
				require.Contains(t, body, `/listen-delete"`)
				require.NotContains(t, body, `/listen-add"`)
			}
			for _, cookie := range []string{"", "invalid-session"} {
				visitor := taskRequest139(a, "GET", path, cookie, nil, nil)
				require.Equal(t, 303, visitor.Code, path)
				require.NotContains(t, visitor.Body.String(), `csrf_token`)
				location, err := url.Parse(visitor.Header().Get("Location"))
				require.NoError(t, err)
				require.Equal(t, path, location.Query().Get("next"))
			}
			head := taskRequest139(a, "HEAD", path, a.cookie, nil, nil)
			require.Equal(t, 200, head.Code, path)
			require.Empty(t, head.Body.String())
		}
	}
	base := "/second/manage/track/12"
	for _, query := range []string{"artist_query=Source&tz=UTC", "merge_query=Target&tz=UTC", "listen_at=100&history_return=%2Fsecond%2Flistens%3Fpage%3D2&tz=America%2FSao_Paulo", "tz=UTC"} {
		w := taskRequest139(a, "GET", base+"?"+query, a.cookie, nil, nil)
		require.Equal(t, 303, w.Code, w.Body.String())
		target, err := url.Parse(w.Header().Get("Location"))
		require.NoError(t, err)
		original, err := url.ParseQuery(query)
		require.NoError(t, err)
		require.Equal(t, original, target.Query())
		require.NotEqual(t, base, target.Path)
		require.Equal(t, 200, taskRequest139(a, "GET", target.String(), a.cookie, nil, nil).Code)
	}
	for _, query := range []string{"listen_at=100&listen_at=100", "tz=NoSuchZone", "unknown=1", "history_return=https%3A%2F%2Fevil.test", "history_return=%2F%2Fevil.test", "history_return=%2Fempty%2Flistens"} {
		require.Equal(t, 400, taskRequest139(a, "GET", base+"/listen?listen_at=100&"+query, a.cookie, nil, nil).Code, query)
	}
	require.Equal(t, 400, taskRequest139(a, "GET", base+"/listen", a.cookie, nil, nil).Code)
	require.Equal(t, 404, taskRequest139(a, "GET", base+"/listen?listen_at=101", a.cookie, nil, nil).Code)
	// Global catalog editing remains available from a profile that has never heard the track.
	require.Equal(t, 200, taskRequest139(a, "GET", "/default/manage/track/12/edit", a.cookie, nil, nil).Code)
	require.Equal(t, 404, taskRequest139(a, "GET", "/default/manage/track/12/listen?listen_at=100", a.cookie, nil, nil).Code)
}

// TestManagementTaskMutationSecurity139 proves rejection precedes mutation and tickets remain one-use.
func TestManagementTaskMutationSecurity139(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	for _, test := range []struct {
		name    string
		status  int
		change  func(url.Values)
		headers map[string]string
		cookie  string
	}{
		{name: "foreign origin", status: 403, headers: map[string]string{"Origin": "https://evil.test"}},
		{name: "cross-site", status: 403, headers: map[string]string{"Sec-Fetch-Site": "cross-site"}},
		{name: "csrf missing", status: 403, change: func(v url.Values) { v.Del("csrf_token") }},
		{name: "csrf wrong", status: 403, change: func(v url.Values) { v.Set("csrf_token", "wrong") }},
		{name: "csrf duplicate", status: 400, change: func(v url.Values) { v.Add("csrf_token", v.Get("csrf_token")) }},
		{name: "ticket missing", status: 409, change: func(v url.Values) { v.Del("submission_token") }},
		{name: "ticket wrong", status: 409, change: func(v url.Values) { v.Set("submission_token", "wrong") }},
		{name: "profile id", status: 409, change: func(v url.Values) { v.Set("profile_id", "3") }},
		{name: "profile slug", status: 409, change: func(v url.Values) { v.Set("profile_slug", "empty") }},
		{name: "external return", status: 400, change: func(v url.Values) { v.Set("return_to", "https://evil.test") }},
		{name: "protocol return", status: 400, change: func(v url.Values) { v.Set("return_to", "//evil.test") }},
		{name: "cross profile return", status: 400, change: func(v url.Values) { v.Set("return_to", "/empty/manage/track/12/edit") }},
		{name: "wrong media return", status: 400, change: func(v url.Values) { v.Set("return_to", "/second/manage/track/13/edit") }},
		{name: "invalid session", status: 401, cookie: "invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			v := taskForm139(t, a, base+"/edit?tz=UTC&artist_query=Source")
			v.Set("alias", "Forbidden Alias")
			if test.change != nil {
				test.change(v)
			}
			cookie := a.cookie
			if test.cookie != "" {
				cookie = test.cookie
			}
			w := taskRequest139(a, "POST", base+"/alias-add", cookie, v, test.headers)
			require.Equal(t, test.status, w.Code, w.Body.String())
			aliases, err := a.store.GetAllTrackAliases(context.Background(), 12)
			require.NoError(t, err)
			require.Len(t, aliases, 1)
		})
	}
	v := taskForm139(t, a, base+"/edit?tz=UTC&artist_query=Source")
	v.Set("alias", "Allowed Alias")
	w := taskRequest139(a, "POST", base+"/alias-add", a.cookie, v, nil)
	require.Equal(t, 303, w.Code, w.Body.String())
	require.Equal(t, base+"/edit?artist_query=Source&tz=UTC#aliases", w.Header().Get("Location"))
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/alias-add", a.cookie, v, nil).Code)
	// Validation keeps the task, entered values and context, and mints a replacement ticket.
	v = taskForm139(t, a, base+"/edit?tz=UTC&artist_query=Source")
	v.Set("mbid", "bad-uuid")
	bad := taskRequest139(a, "POST", base+"/metadata", a.cookie, v, nil)
	require.Equal(t, 400, bad.Code)
	require.Contains(t, bad.Body.String(), `value="bad-uuid"`)
	require.NotContains(t, bad.Body.String(), `/listen-add"`)
	require.NotEqual(t, v.Get("submission_token"), taskField139(t, bad, "submission_token"))
	// A ticket from another session cannot authorize this session even with its valid CSRF.
	user, err := a.store.GetAdminUser(context.Background())
	require.NoError(t, err)
	session, err := a.store.SaveSession(context.Background(), user.ID, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	other := taskRequest139(a, "GET", base+"/edit", session.ID.String(), nil, nil)
	v = taskForm139(t, a, base+"/edit")
	v.Set("submission_token", taskField139(t, other, "submission_token"))
	v.Set("alias", "Other session")
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/alias-add", a.cookie, v, nil).Code)
	// Deleting one personal listen never deletes the same track/instant in another profile.
	v = taskForm139(t, a, base+"/listen?listen_at=100&tz=UTC")
	v.Set("unix", "100")
	v.Set("confirm", "1")
	w = taskRequest139(a, "POST", base+"/listen-delete", a.cookie, v, nil)
	require.Equal(t, 303, w.Code, w.Body.String())
	require.Contains(t, w.Header().Get("Location"), base+"/listens")
	require.Equal(t, 404, taskRequest139(a, "GET", base+"/listen?listen_at=100", a.cookie, nil, nil).Code)
	require.Equal(t, 200, taskRequest139(a, "GET", "/empty/manage/track/12/listen?listen_at=100", a.cookie, nil, nil).Code)
	count, err := a.store.ProfileListenCount(context.Background(), 3)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

// TestManagementCatalogDestinations139 verifies normal and reversed survivor redirects and delete fallback.
func TestManagementCatalogDestinations139(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "reversed"}[reverse], func(t *testing.T) {
			a := taskApp139(t)
			base := "/second/manage/track/12"
			query := "?action=merge&to=13"
			want := "/second/track/13"
			if reverse {
				query += "&reverse=1"
				want = "/second/track/12"
			}
			review := taskRequest139(a, "GET", base+"/review"+query, a.cookie, nil, nil)
			require.Equal(t, 200, review.Code)
			v := url.Values{"csrf_token": {taskField139(t, review, "csrf_token")}, "plan": {taskField139(t, review, "plan")}, "profile_id": {"2"}, "profile_slug": {"second"}, "confirm": {"1"}}
			// Review retains its dedicated plan token, rather than changing the mutation contract.
			require.Equal(t, 403, taskRequest139(a, "POST", base+"/confirm", a.cookie, v, map[string]string{"Origin": "https://evil.test"}).Code)
			out := taskRequest139(a, "POST", base+"/confirm", a.cookie, v, nil)
			require.Equal(t, 303, out.Code)
			require.Equal(t, want, out.Header().Get("Location"))
			require.Equal(t, 409, taskRequest139(a, "POST", base+"/confirm", a.cookie, v, nil).Code)
			target := want[strings.LastIndex(want, "/")+1:]
			review = taskRequest139(a, "GET", "/second/manage/track/"+target+"/review?action=delete", a.cookie, nil, nil)
			require.Equal(t, 200, review.Code)
			v.Set("plan", taskField139(t, review, "plan"))
			v.Set("csrf_token", taskField139(t, review, "csrf_token"))
			// Personal deletion requires a submission ticket and preserves the survivor catalog row.
			v.Set("submission_token", taskField139(t, review, "submission_token"))
			out = taskRequest139(a, "POST", "/second/manage/track/"+target+"/confirm", a.cookie, v, nil)
			require.Equal(t, 303, out.Code)
			require.Equal(t, "/second", out.Header().Get("Location"))
			_, err := a.store.GetTrack(context.Background(), db.GetTrackOpts{ID: map[string]int32{"12": 12, "13": 13}[target]})
			require.NoError(t, err)
		})
	}
}

// TestManagementTaskFragments139 keeps the ISSUE-124 markers, ticket rotation and search semantics.
func TestManagementTaskFragments139(t *testing.T) {
	a := taskApp139(t)
	base := "/second/manage/track/12"
	v := taskForm139(t, a, base+"/edit?artist_query=Source&tz=UTC")
	headers := map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Submission": v.Get("submission_token")}
	for _, task := range []string{"edit", "merge"} {
		query, section := "artist_query=Source", "artists"
		if task == "merge" {
			query, section = "merge_query=Target", "merge"
		}
		path := base + "/" + task + "?" + query + "&tz=UTC&partial=1"
		fragment := taskRequest139(a, "GET", path, a.cookie, nil, headers)
		require.Equal(t, 200, fragment.Code, fragment.Body.String())
		require.Equal(t, section, fragment.Header().Get("X-TuneTrail-Fragment"))
		require.Equal(t, v.Get("submission_token"), fragment.Header().Get("X-TuneTrail-Submission"))
		require.NotContains(t, fragment.Body.String(), "<html")
		navigation := taskRequest139(a, "GET", path, a.cookie, nil, nil)
		require.Equal(t, 303, navigation.Code)
		require.NotContains(t, navigation.Header().Get("Location"), "partial")
		require.Contains(t, navigation.Header().Get("Location"), base+"/"+task)
	}
	require.Equal(t, 400, taskRequest139(a, "GET", base+"/image?artist_query=Source&partial=1", a.cookie, nil, headers).Code)
	v = taskForm139(t, a, base+"/add-listen?tz=UTC")
	v.Set("local_time", "2024-01-02T03:04")
	v.Set("timezone", "UTC")
	v.Set("offset", "")
	fragment := taskRequest139(a, "POST", base+"/listen-add", a.cookie, v, headers)
	require.Equal(t, 200, fragment.Code, fragment.Body.String())
	require.Equal(t, "listens", fragment.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, fragment.Body.String(), `/listen-add"`)
	require.NotContains(t, fragment.Body.String(), `/listen-delete"`)
	require.Contains(t, fragment.Body.String(), "Listen added.")
	require.NotEqual(t, v.Get("submission_token"), fragment.Header().Get("X-TuneTrail-Submission"))
	require.Equal(t, 409, taskRequest139(a, "POST", base+"/listen-add", a.cookie, v, headers).Code)
	require.Equal(t, 401, taskRequest139(a, "POST", base+"/listen-add", "", v, nil).Code)
}

// TestManagementTaskLogin139 logs in through the router and returns to the exact task context.
func TestManagementTaskLogin139(t *testing.T) {
	a := taskApp139(t)
	next := "/second/manage/track/12/listen?listen_at=100&tz=UTC&history_return=%2Fsecond%2Flistens"
	visitor := taskRequest139(a, "GET", next, "", nil, nil)
	login := taskRequest139(a, "GET", visitor.Header().Get("Location"), "", nil, nil)
	require.Equal(t, 200, login.Code)
	require.Equal(t, next, taskField139(t, login, "next"))
	form := url.Values{"csrf_token": {taskField139(t, login, "csrf_token")}, "username": {"administrator"}, "password": {"test-password"}, "next": {next}}
	r := httptest.NewRequest("POST", "http://koito.test/admin/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range login.Result().Cookies() {
		r.AddCookie(cookie)
	}
	out := httptest.NewRecorder()
	a.router.ServeHTTP(out, r)
	require.Equal(t, 303, out.Code, out.Body.String())
	require.Equal(t, next, out.Header().Get("Location"))
}

// TestManagementHistoryMediaReturn139 preserves an old history link from a media page.
func TestManagementHistoryMediaReturn139(t *testing.T) {
	a := taskApp139(t)
	back := "/second/track/12?tz=UTC"
	old := "/second/manage/track/12?listen_at=100&tz=UTC&history_return=" + url.QueryEscape(back)
	legacy := taskRequest139(a, "GET", old, a.cookie, nil, nil)
	require.Equal(t, 303, legacy.Code)
	form := taskForm139(t, a, legacy.Header().Get("Location"))
	require.Equal(t, back, form.Get("return_to"))
	form.Set("unix", "100")
	form.Set("confirm", "1")
	out := taskRequest139(a, "POST", "/second/manage/track/12/listen-delete", a.cookie, form, nil)
	require.Equal(t, 303, out.Code, out.Body.String())
	require.Equal(t, "/second/manage/track/12?tz=UTC", out.Header().Get("Location"))
	redirect := taskRequest139(a, "GET", out.Header().Get("Location"), a.cookie, nil, nil)
	require.Equal(t, 303, redirect.Code)
	require.Equal(t, 200, taskRequest139(a, "GET", redirect.Header().Get("Location"), a.cookie, nil, nil).Code)
}

// TestManagementUnplayedSurvivor139 keeps global editing usable without widening public reads.
func TestManagementUnplayedSurvivor139(t *testing.T) {
	for _, kind := range []string{"track", "album", "artist"} {
		for _, slug := range []string{"second", "default"} {
			t.Run(kind+"/"+slug, func(t *testing.T) {
				a := taskApp139(t)
				require.NoError(t, a.store.Exec(`INSERT INTO releases(id,various_artists) VALUES(21,0)`))
				require.NoError(t, a.store.Exec(`INSERT INTO release_aliases VALUES(21,'Target Album','User',1)`))
				from, to, profileID := "12", "13", "2"
				if kind == "album" {
					from, to = "11", "21"
				}
				if kind == "artist" {
					from, to = "10", "20"
				}
				if slug == "default" {
					profileID = "1"
				}
				base := "/" + slug + "/manage/" + kind + "/" + from
				review := taskRequest139(a, "GET", base+"/review?action=merge&to="+to, a.cookie, nil, nil)
				require.Equal(t, 200, review.Code, review.Body.String())
				form := url.Values{"csrf_token": {taskField139(t, review, "csrf_token")}, "plan": {taskField139(t, review, "plan")}, "profile_id": {profileID}, "profile_slug": {slug}, "confirm": {"1"}}
				out := taskRequest139(a, "POST", base+"/confirm", a.cookie, form, nil)
				require.Equal(t, 303, out.Code, out.Body.String())
				want := "/" + slug + "/" + kind + "/" + to
				if slug == "default" {
					want = "/default/manage/" + kind + "/" + to
				}
				require.Equal(t, want, out.Header().Get("Location"))
				require.Equal(t, 200, taskRequest139(a, "GET", want, a.cookie, nil, nil).Code)
				if slug == "default" {
					require.Equal(t, 404, taskRequest139(a, "GET", "/default/"+kind+"/"+to, a.cookie, nil, nil).Code)
				}
			})
		}
	}
}

// TestManagementLegacyListenDeleteReturn139 clears obsolete selection context from saved forms.
func TestManagementLegacyListenDeleteReturn139(t *testing.T) {
	for _, task := range []string{"", "/listens", "/edit"} {
		t.Run(task, func(t *testing.T) {
			a := taskApp139(t)
			base := "/second/manage/track/12"
			form := taskForm139(t, a, base+"/listen?listen_at=100&tz=UTC")
			form.Set("return_to", base+task+"?listen_at=100&tz=UTC")
			form.Set("unix", "100")
			form.Set("confirm", "1")
			out := taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, nil)
			require.Equal(t, 303, out.Code, out.Body.String())
			target := out.Header().Get("Location")
			require.NotContains(t, target, "listen_at")
			require.Contains(t, target, "tz=UTC")
			require.Equal(t, 200, taskRequest139(a, "GET", target, a.cookie, nil, nil).Code)
			require.Equal(t, 200, taskRequest139(a, "GET", "/empty/manage/track/12/listen?listen_at=100", a.cookie, nil, nil).Code)
		})
	}
}

// TestManagementListenErrorReturn139 retains validation feedback when returning to filtered history.
func TestManagementListenErrorReturn139(t *testing.T) {
	a := taskApp139(t)
	back := "/second/listens?year=2024&week=44&page=2&limit=1&track_id=12&tz=UTC"
	base := "/second/manage/track/12"
	form := taskForm139(t, a, base+"/listen?listen_at=100&history_return="+url.QueryEscape(back))
	require.Equal(t, back, form.Get("return_to"))
	form.Set("unix", "100") // Missing confirmation is a form error, not a lost history context.
	out := taskRequest139(a, "POST", base+"/listen-delete", a.cookie, form, nil)
	require.Equal(t, 400, out.Code)
	require.Contains(t, out.Body.String(), `id="listens"`)
	require.Contains(t, out.Body.String(), html.EscapeString(back))
	require.NotEmpty(t, taskField139(t, out, "csrf_token"))
	require.NotEqual(t, form.Get("submission_token"), taskField139(t, out, "submission_token"))
	require.Equal(t, 200, taskRequest139(a, "GET", base+"/listen?listen_at=100", a.cookie, nil, nil).Code)
}
