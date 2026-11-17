package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"

	"github.com/stretchr/testify/require"
)

// adminActions returns the role-aware group inside More (ISSUE-156, migrated from ISSUE-123).
func adminActions(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `class="nav-admin-actions"`)
	require.GreaterOrEqual(t, start, 0, "administration actions missing")
	return body[start : start+strings.Index(body[start:], "</section>")]
}

// TestAdminActionsItemsByRole checks which administration links each audience receives.
func TestAdminActionsItemsByRole(t *testing.T) {
	a := newProfileTestApp(t)
	body := a.request(http.MethodGet, "/default", "", "", false).Body.String()
	visitor := adminActions(t, body)
	require.Contains(t, body, `href="https://github.com/ebrgr/TuneTrail"`)
	require.Contains(t, body, `rel="noopener"`)
	require.Contains(t, visitor, `href="/admin"`)
	require.NotContains(t, visitor, "/admin/settings")
	require.NotContains(t, visitor, "section=account")

	admin := adminActions(t, a.request(http.MethodGet, "/default", "", "", true).Body.String())
	// Administration links keep the current profile so its header stays in view.
	for _, link := range []string{`href="/admin?section=account&profile=1"`, `href="/admin?section=profiles&profile=1"`,
		`href="/admin/settings?profile=1"`} {
		require.Contains(t, admin, link)
	}
}

// TestAdminActionsCurrentInAdministration checks selected links in the native disclosure.
func TestAdminActionsCurrentInAdministration(t *testing.T) {
	a := newProfileTestApp(t)
	settings := adminActions(t, a.request(http.MethodGet, "/admin/settings", "", "", true).Body.String())
	require.Regexp(t, `href="/admin/settings"\s+aria-current="page"`, settings)

	account := adminActions(t, a.request(http.MethodGet, "/admin?section=account", "", "", true).Body.String())
	require.Regexp(t, `href="/admin\?section=account"\s+aria-current="page"`, account)

	profiles := adminActions(t, a.request(http.MethodGet, "/admin?section=profiles", "", "", true).Body.String())
	require.Regexp(t, `href="/admin\?section=profiles"\s+aria-current="page"`, profiles)

	login := adminActions(t, a.request(http.MethodGet, "/admin", "", "", false).Body.String())
	require.NotContains(t, login, "/admin/settings", "the login page offers only Sign in and About")
}

// TestAdminActionsPreAuthErrorShowsVisitorItems guards the role source: an error raised
// before authentication (login rate limit) must not offer administrator links (ISSUE-123).
func TestAdminActionsPreAuthErrorShowsVisitorItems(t *testing.T) {
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
	bar := adminActions(t, last.Body.String())
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
