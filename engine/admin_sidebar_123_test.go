package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"

	"github.com/stretchr/testify/require"
)

// secondBar returns the rendered second sidebar (ISSUE-123).
func secondBar(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `id="nav-admin"`)
	require.GreaterOrEqual(t, start, 0, "second sidebar missing")
	return body[start : start+strings.Index(body[start:], "</nav>")]
}

// TestSecondSidebarItemsByRole checks which administration links each audience receives.
func TestSecondSidebarItemsByRole(t *testing.T) {
	a := newProfileTestApp(t)
	visitor := secondBar(t, a.request(http.MethodGet, "/default", "", "", false).Body.String())
	require.Contains(t, visitor, "hidden\n")
	require.Contains(t, visitor, `href="/admin"`)
	require.NotContains(t, visitor, "/admin/settings")
	require.NotContains(t, visitor, "section=account")

	admin := secondBar(t, a.request(http.MethodGet, "/default", "", "", true).Body.String())
	require.Contains(t, admin, "hidden\n", "public pages start with the second sidebar closed")
	// Administration links keep the current profile so its header stays in view.
	for _, link := range []string{`href="/admin?section=account&profile=1"`, `href="/admin?section=profiles&profile=1"`,
		`href="/admin/settings?profile=1"`, `href="https://github.com/ebrgr/TuneTrail"`} {
		require.Contains(t, admin, link)
	}
}

// TestSecondSidebarOpenInAdministration checks the open state and the current item.
func TestSecondSidebarOpenInAdministration(t *testing.T) {
	a := newProfileTestApp(t)
	settings := secondBar(t, a.request(http.MethodGet, "/admin/settings", "", "", true).Body.String())
	require.NotContains(t, settings, "hidden\n")
	require.Regexp(t, `href="/admin/settings"\s+aria-current="page"`, settings)

	account := secondBar(t, a.request(http.MethodGet, "/admin?section=account", "", "", true).Body.String())
	require.Regexp(t, `href="/admin\?section=account"\s+aria-current="page"`, account)

	login := secondBar(t, a.request(http.MethodGet, "/admin", "", "", false).Body.String())
	require.NotContains(t, login, "/admin/settings", "the login page offers only Sign in and About")
}

// TestSecondSidebarPreAuthErrorShowsVisitorItems guards the role source: an error raised
// before authentication (login rate limit) must not offer administrator links (ISSUE-123).
func TestSecondSidebarPreAuthErrorShowsVisitorItems(t *testing.T) {
	a := newProfileTestApp(t)
	h, err := adminui.New(a.store, adminui.Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	var last *httptest.ResponseRecorder
	for i := 0; i < 12; i++ {
		r := httptest.NewRequest(http.MethodPost, "/admin/login", strings.NewReader("username=x&password=y"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		last = httptest.NewRecorder()
		h.ServeHTTP(last, r)
	}
	require.Equal(t, http.StatusTooManyRequests, last.Code)
	bar := secondBar(t, last.Body.String())
	require.Contains(t, bar, `href="/admin"`)
	require.NotContains(t, bar, "/admin/settings")
	require.NotContains(t, bar, "section=profiles")
}

// TestAdministrationPagesShowProfileHeader keeps the public header pattern in administration.
func TestAdministrationPagesShowProfileHeader(t *testing.T) {
	a := newProfileTestApp(t)
	for _, path := range []string{"/admin/settings?profile=2", "/admin?section=account&profile=2", "/admin?profile=2"} {
		w := a.request(http.MethodGet, path, "", "", true)
		require.Equal(t, http.StatusOK, w.Code, path)
		body := w.Body.String()
		require.Contains(t, body, `<header class="profile-header">`, path)
		require.Contains(t, body, "<strong>Second</strong>", path)
		require.Contains(t, body, `class="admin-badge"`, path)
	}
	require.Equal(t, http.StatusBadRequest, a.request(http.MethodGet, "/admin/settings?other=1", "", "", true).Code)
	require.NotContains(t, a.request(http.MethodGet, "/admin/settings", "", "", true).Body.String(), `<header class="profile-header">`)
}
