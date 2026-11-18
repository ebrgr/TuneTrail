package engine

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminBadgeOnlyForAdministratorSession guards ISSUE-109: the badge and data-admin are
// rendered from a valid administrator session only, never for visitors or bad cookies.
func TestAdminBadgeOnlyForAdministratorSession(t *testing.T) {
	a := newProfileTestApp(t)
	page := func(cookie string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "http://koito.test/default", nil)
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: cookie})
		}
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "private, no-cache", w.Header().Get("Cache-Control"))
		require.Contains(t, w.Header().Values("Vary"), "Cookie")
		return w
	}

	admin := page(a.cookie)
	require.Equal(t, "true", admin.Header().Get("X-TuneTrail-Admin"))
	require.Contains(t, admin.Body.String(), `class="admin-badge"`)
	require.Contains(t, admin.Body.String(), "data-admin\n")
	require.Contains(t, admin.Body.String(), `href="/admin?profile=1"`) // badge keeps the profile (ISSUE-123)

	for name, cookie := range map[string]string{
		"visitor":     "",
		"garbage":     "not-a-uuid",
		"unknown":     "00000000-0000-4000-8000-000000000001",
		"nil session": "00000000-0000-0000-0000-000000000000",
	} {
		w := page(cookie)
		require.Equal(t, "false", w.Header().Get("X-TuneTrail-Admin"), name)
		require.NotContains(t, w.Body.String(), "admin-badge", name)
		require.NotContains(t, w.Body.String(), "data-admin", name)
		require.False(t, strings.Contains(w.Body.String(), `data-tip="Administrator`), name)
	}
}

// TestAdminBadgeHeadAndAmbiguousCookie checks HEAD headers and rejects duplicated cookies.
func TestAdminBadgeHeadAndAmbiguousCookie(t *testing.T) {
	a := newProfileTestApp(t)
	head := a.request(http.MethodHead, "/default", "", "", true)
	require.Equal(t, http.StatusOK, head.Code)
	require.Equal(t, "true", head.Header().Get("X-TuneTrail-Admin"))
	require.Equal(t, "private, no-cache", head.Header().Get("Cache-Control"))
	require.Contains(t, head.Header().Values("Vary"), "Cookie")

	r := httptest.NewRequest(http.MethodGet, "http://koito.test/default", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, "false", w.Header().Get("X-TuneTrail-Admin"))
	require.NotContains(t, w.Body.String(), "admin-badge")
}

// TestAdminBadgeRequiresAdministratorRole rejects a valid session of a non-admin user.
func TestAdminBadgeRequiresAdministratorRole(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`UPDATE users SET role='admin',status='active',password=X'01' WHERE id=2; UPDATE users SET role='user' WHERE id=1`))
	w := a.request(http.MethodGet, "/default", "", "", true)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "false", w.Header().Get("X-TuneTrail-Admin"))
	require.NotContains(t, w.Body.String(), "admin-badge")
}

// TestAdminBadgeDisappearsAfterLogout checks that an ended session renders a visitor page.
func TestAdminBadgeDisappearsAfterLogout(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`DELETE FROM sessions`))
	w := a.request(http.MethodGet, "/default", "", "", true)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "false", w.Header().Get("X-TuneTrail-Admin"))
	require.NotContains(t, w.Body.String(), "admin-badge")
}
