package engine

// Session cookie rename (ISSUE-111, phase B2): only tunetrail_session is accepted.
// By user decision there is no koito_session fallback; administrators sign in
// again once after upgrading.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// asUser requests /apis/web/v1/user with a raw Cookie header.
func asUser(a *profileTestApp, cookie string) int {
	r := httptest.NewRequest("GET", "http://koito.test/apis/web/v1/user", nil)
	r.Header.Set("Cookie", cookie)
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	return w.Code
}

func TestOnlyCurrentSessionCookieAuthenticates(t *testing.T) {
	a := newProfileTestApp(t)
	require.Equal(t, http.StatusOK, asUser(a, "tunetrail_session="+a.cookie))
	require.Equal(t, http.StatusUnauthorized, asUser(a, "koito_session="+a.cookie), "the legacy cookie must be ignored")
}

func TestLoginSetsOnlyCurrentCookie(t *testing.T) {
	a := newProfileTestApp(t)
	r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/login", strings.NewReader(`{"username":"administrator","password":"test-password"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, http.StatusNoContent, w.Code)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "tunetrail_session", cookies[0].Name)
	require.NotContains(t, strings.Join(w.Result().Header.Values("Set-Cookie"), ";"), "koito")
}
