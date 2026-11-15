package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// searchFragment requests the instant search fragment like search.js does (ISSUE-107).
func searchFragment(a *profileTestApp, path string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "http://koito.test"+path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	return w
}

// TestInstantSearchFragment covers the fragment contract: grouping, limit, hints,
// headers and refusal outside the search dialog.
func TestInstantSearchFragment(t *testing.T) {
	a := newProfileTestApp(t)
	for i := 1; i <= 7; i++ {
		for _, q := range []string{
			fmt.Sprintf(`INSERT INTO artists(id) VALUES(%d)`, 100+i),
			fmt.Sprintf(`INSERT INTO artist_aliases VALUES(%d,'Searchable Artist %d','User',1)`, 100+i, i),
			fmt.Sprintf(`INSERT INTO releases(id,various_artists) VALUES(%d,0)`, 200+i),
			fmt.Sprintf(`INSERT INTO release_aliases VALUES(%d,'Other Album %d','User',1)`, 200+i, i),
			fmt.Sprintf(`INSERT INTO tracks(id,release_id,duration) VALUES(%d,%d,60)`, 300+i, 200+i),
			fmt.Sprintf(`INSERT INTO track_aliases VALUES(%d,'Other Track %d','User',1)`, 300+i, i),
			fmt.Sprintf(`INSERT INTO artist_tracks VALUES(%d,%d,1)`, 100+i, 300+i),
			fmt.Sprintf(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,%d,%d)`, 300+i, 1000+i),
		} {
			require.NoError(t, a.store.Exec(q))
		}
	}
	partial := map[string]string{"X-TuneTrail-Partial": "1"}

	w := searchFragment(a, "/default/search?q=Searchable&partial=1", partial)
	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	require.NotContains(t, body, "<html")
	require.Contains(t, body, `role="listbox"`)
	require.Equal(t, 5, strings.Count(body, `role="option"`), "at most five items per kind")
	require.Contains(t, body, "7 results")
	require.Contains(t, body, `href="/default/search?q=Searchable"`)
	require.Equal(t, "private, no-cache", w.Header().Get("Cache-Control"))
	require.Contains(t, w.Header().Values("Vary"), "X-TuneTrail-Partial")
	require.Equal(t, "noindex", w.Header().Get("X-Robots-Tag"))

	require.Contains(t, searchFragment(a, "/default/search?q=S&partial=1", partial).Body.String(), "at least 2 characters")
	require.Contains(t, searchFragment(a, "/default/search?q=zzzz&partial=1", partial).Body.String(), "No results.")
	require.Equal(t, http.StatusBadRequest, searchFragment(a, "/default/search?q="+strings.Repeat("x", 201)+"&partial=1", partial).Code)
	require.Equal(t, http.StatusNotFound, searchFragment(a, "/missing/search?q=Searchable&partial=1", partial).Code)

	// A navigation to partial=1 (typed, bookmarked or cross-site) gets the full page instead.
	for _, h := range []map[string]string{nil, {"X-TuneTrail-Partial": "1", "Sec-Fetch-Mode": "navigate"}} {
		w := searchFragment(a, "/default/search?q=Searchable&partial=1", h)
		require.Equal(t, http.StatusSeeOther, w.Code)
		require.Equal(t, "/default/search?q=Searchable", w.Header().Get("Location"))
	}
}
