package engine

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestMediaSectionsWithoutReload covers ISSUE-124 with single-use tickets enforced: anchors
// without JavaScript, section fragments that chain through fresh tickets, refusals that
// never answer with a fragment, and search fragments that do not mint tickets.
func TestMediaSectionsWithoutReload(t *testing.T) {
	a := newProfileTestApp(t)
	for _, q := range []string{`INSERT INTO artists(id) VALUES(10)`, `INSERT INTO artist_aliases VALUES(10,'Fictional Artist','User',1)`,
		`INSERT INTO releases(id,various_artists) VALUES(11,0)`, `INSERT INTO release_aliases VALUES(11,'Fictional Album','User',1)`,
		`INSERT INTO artist_releases VALUES(10,11,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(12,11,60)`,
		`INSERT INTO track_aliases VALUES(12,'Fictional Track','User',1)`, `INSERT INTO artist_tracks VALUES(10,12,1)`,
		`INSERT INTO artists(id) VALUES(20)`, `INSERT INTO artist_aliases VALUES(20,'Fictional Guest','User',1)`} {
		require.NoError(t, a.store.Exec(q))
	}
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir(), RequireSubmissionToken: true})
	require.NoError(t, err)
	call := func(method, path string, values url.Values, headers map[string]string) *httptest.ResponseRecorder {
		body := ""
		if values != nil {
			body = values.Encode()
		}
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if values != nil {
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	field := func(body, name string) string {
		m := regexp.MustCompile(`name="` + name + `" value="([^"]+)"`).FindStringSubmatch(body)
		require.Len(t, m, 2, name)
		return m[1]
	}
	partial := map[string]string{"X-TuneTrail-Partial": "1"}
	base := "/second/manage/track/12"
	page := call("GET", base, nil, nil).Body.String()
	csrf, ticket := field(page, "csrf_token"), field(page, "submission_token")
	form := func(extra url.Values) url.Values {
		v := url.Values{"csrf_token": {csrf}, "submission_token": {ticket}, "profile_id": {"2"}, "profile_slug": {"second"}}
		for k, vs := range extra {
			v[k] = vs
		}
		return v
	}

	// Without JavaScript the redirect returns to the edited section.
	out := call("POST", base+"/alias-add", form(url.Values{"alias": {"First Alias"}}), nil)
	require.Equal(t, http.StatusSeeOther, out.Code)
	require.Equal(t, base+"#aliases", out.Header().Get("Location"))

	// With JavaScript: only the section, plus a fresh ticket that the next action can use.
	ticket = field(call("GET", base, nil, nil).Body.String(), "submission_token")
	out = call("POST", base+"/alias-add", form(url.Values{"alias": {"Second Alias"}}), partial)
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	require.Equal(t, "aliases", out.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, "no-store", out.Header().Get("Cache-Control"))
	require.NotContains(t, out.Body.String(), "<html")
	require.Contains(t, out.Body.String(), `id="aliases"`)
	require.Contains(t, out.Body.String(), "Second Alias")
	spent, fresh := ticket, out.Header().Get("X-TuneTrail-Submission")
	_, err = uuid.Parse(fresh)
	require.NoError(t, err)
	require.Equal(t, fresh, field(out.Body.String(), "submission_token"))

	// The spent ticket is refused with a full error page, never with a fragment.
	ticket = spent
	out = call("POST", base+"/metadata", form(url.Values{"mbid": {""}, "various": {""}}), partial)
	require.Equal(t, http.StatusConflict, out.Code)
	require.Empty(t, out.Header().Get("X-TuneTrail-Fragment"))

	// The fresh ticket works for an action in another section.
	ticket = fresh
	out = call("POST", base+"/alias-primary", form(url.Values{"alias": {"Second Alias"}}), partial)
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	require.Contains(t, out.Body.String(), `data-heading="Manage track: Second Alias"`)
	ticket = out.Header().Get("X-TuneTrail-Submission")

	// Validation errors stay inside the section, with the section's status.
	out = call("POST", base+"/metadata", form(url.Values{"mbid": {"not-a-uuid"}, "various": {""}}), partial)
	require.Equal(t, http.StatusBadRequest, out.Code)
	require.Equal(t, "metadata", out.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, out.Body.String(), `data-section-message`)
	ticket = out.Header().Get("X-TuneTrail-Submission")

	// Notices of listen actions are shown in the listens section right away.
	out = call("POST", base+"/listen-add", form(url.Values{"local_time": {"2023-11-14T19:13"}, "timezone": {"UTC"}, "offset": {""}}), partial)
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	require.Equal(t, "listens", out.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, out.Body.String(), "Listen added.")
	ticket = out.Header().Get("X-TuneTrail-Submission")

	// CSRF failures never answer with a fragment.
	bad := form(url.Values{"alias": {"Nope"}})
	bad.Set("csrf_token", "wrong")
	out = call("POST", base+"/alias-add", bad, partial)
	require.Equal(t, http.StatusForbidden, out.Code)
	require.Empty(t, out.Header().Get("X-TuneTrail-Fragment"))

	// Search fragments reuse the page ticket instead of minting new ones.
	out = call("GET", base+"?artist_query=Fictional&tz=UTC&partial=1", nil, map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Submission": ticket})
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	require.Equal(t, "artists", out.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, ticket, out.Header().Get("X-TuneTrail-Submission"))
	require.Contains(t, out.Body.String(), "Add Fictional Artist")
	// The search context (tz and query) survives in the section's forms (Reviewer).
	returnTo := field(out.Body.String(), "return_to")
	require.Contains(t, returnTo, "artist_query=Fictional")
	require.Contains(t, returnTo, "tz=UTC")
	require.NotContains(t, returnTo, "partial")

	// Adding an artist from the results keeps the result list in the returned section.
	add := form(url.Values{"artist_id": {"20"}})
	add.Set("return_to", strings.ReplaceAll(returnTo, "&amp;", "&"))
	out = call("POST", base+"/artist-add", add, partial)
	require.Equal(t, http.StatusOK, out.Code, out.Body.String())
	require.Equal(t, "artists", out.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, out.Body.String(), "Add Fictional Artist", "search results kept after an enhanced artist action")
	require.Contains(t, out.Body.String(), "Remove Fictional Guest from track")
	ticket = out.Header().Get("X-TuneTrail-Submission")
	out = call("GET", base+"?merge_query=Fictional&partial=1", nil, map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Submission": "not-a-uuid"})
	require.Equal(t, "merge", out.Header().Get("X-TuneTrail-Fragment"))
	require.Empty(t, out.Header().Get("X-TuneTrail-Submission"))

	// A navigation to partial=1 gets the full page without it.
	out = call("GET", base+"?merge_query=Fictional&partial=1", nil, nil)
	require.Equal(t, http.StatusSeeOther, out.Code)
	require.Equal(t, base+"?merge_query=Fictional", out.Header().Get("Location"))
}
