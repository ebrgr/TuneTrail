package engine

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// TestUserAuthenticationKeepsAdminRoutesClosed159 exercises the production HTML and JSON boundaries.
func TestUserAuthenticationKeepsAdminRoutesClosed159(t *testing.T) {
	a := newProfileTestApp(t)
	_, err := a.store.SaveUser(context.Background(), db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	login := a.request("POST", "/apis/web/v1/login", `{"username":"listener","password":"test-password"}`, "", false)
	require.Equal(t, 204, login.Code)
	cookie := login.Result().Cookies()[0]
	for _, path := range []string{"/admin", "/admin/settings", "/admin/profile/1/delete-review", "/admin/profile/1/export", "/apis/web/v1/user", "/apis/web/v1/profiles/default/management"} {
		r := httptest.NewRequest("GET", "http://koito.test"+path, nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, 403, w.Code, path)
	}
	r := httptest.NewRequest("POST", "http://koito.test/admin/profile/1/save", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
	// The JSON logout path is already available to either account role.
	r = httptest.NewRequest("POST", "http://koito.test/apis/web/v1/logout", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, 204, w.Code)
	_, err = admin.Login(context.Background(), a.store, "listener", "wrong-password", false)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
}

// TestSharedLoginLimiter159 proves HTML and JSON still share the original ten-attempt budget.
func TestSharedLoginLimiter159(t *testing.T) {
	a := newProfileTestApp(t)
	for i := 0; i < 10; i++ {
		path := "/apis/web/v1/login"
		if i%2 == 1 {
			path = "/admin/login"
		}
		// Invalid bodies still count across endpoints; the budget is not per account or IP.
		r := httptest.NewRequest("POST", "http://koito.test"+path, strings.NewReader(`{}`))
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.NotEqual(t, http.StatusTooManyRequests, w.Code)
	}
	for _, path := range []string{"/apis/web/v1/login", "/admin/login"} {
		r := httptest.NewRequest("POST", "http://koito.test"+path, strings.NewReader(`{"username":"another-account"}`))
		r.RemoteAddr = "198.51.100.2:9876"
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, http.StatusTooManyRequests, w.Code, path)
	}
}

// TestUserHTMLLoginAndLogout159 prevents a non-admin cookie from trapping a browser at a forbidden page.
func TestUserHTMLLoginAndLogout159(t *testing.T) {
	a := taskApp139(t)
	_, err := a.store.SaveUser(context.Background(), db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	loginPage := taskRequest139(a, "GET", "/admin", "", nil, nil)
	form := url.Values{"csrf_token": {taskField139(t, loginPage, "csrf_token")}, "username": {"listener"}, "password": {"test-password"}}
	r := httptest.NewRequest("POST", "http://koito.test/admin/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range loginPage.Result().Cookies() {
		r.AddCookie(cookie)
	}
	logged := httptest.NewRecorder()
	a.router.ServeHTTP(logged, r)
	require.Equal(t, 303, logged.Code)
	var userCookie string
	for _, cookie := range logged.Result().Cookies() {
		if cookie.Name == "tunetrail_session" {
			userCookie = cookie.Value
		}
	}
	require.NotEmpty(t, userCookie)
	for _, tc := range []struct{ locale, message, button string }{
		{"en", "This account cannot access administration. Sign out to use another account.", "Sign out"},
		{"pt-BR", "Esta conta não pode acessar a administração. Saia para usar outra conta.", "Sair"},
	} {
		page := taskRequest139(a, "GET", "/admin", userCookie, nil, map[string]string{"Cookie": "tunetrail_session=" + userCookie + "; tunetrail_language=" + tc.locale})
		require.Equal(t, 403, page.Code)
		require.Contains(t, page.Body.String(), tc.message)
		require.Contains(t, page.Body.String(), tc.button)
		require.Contains(t, page.Body.String(), `action="/admin/logout"`)
		for _, secret := range []string{`href="/admin/settings"`, `action="/admin/profile/create"`, `action="/admin/account/password"`, `href="/admin?profile=`, "Source Track"} {
			require.NotContains(t, page.Body.String(), secret)
		}
	}
	page := taskRequest139(a, "GET", "/admin", userCookie, nil, nil)
	logout := url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")}}
	require.Equal(t, 403, taskRequest139(a, "POST", "/admin/logout", userCookie, url.Values{}, nil).Code)
	other := taskRequest139(a, "GET", "/admin", a.cookie, nil, nil)
	wrong := url.Values{"csrf_token": {taskField139(t, other, "csrf_token")}, "submission_token": {logout.Get("submission_token")}}
	require.Equal(t, 403, taskRequest139(a, "POST", "/admin/logout", userCookie, wrong, nil).Code)
	require.Equal(t, 403, taskRequest139(a, "POST", "/admin/logout", userCookie, logout, map[string]string{"Origin": "http://evil.test"}).Code)
	require.Equal(t, 404, taskRequest139(a, "GET", "/admin/logout", userCookie, nil, nil).Code)
	out := taskRequest139(a, "POST", "/admin/logout", userCookie, logout, nil)
	require.Equal(t, 303, out.Code)
	require.Equal(t, "/admin", out.Header().Get("Location"))
	userID, err := uuid.Parse(userCookie)
	require.NoError(t, err)
	_, err = admin.Authenticate(context.Background(), a.store, userID)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	require.Equal(t, 401, taskRequest139(a, "POST", "/admin/logout", userCookie, logout, nil).Code)
	next := taskRequest139(a, "GET", "/admin", userCookie, nil, nil)
	require.Equal(t, 200, next.Code)
	require.Contains(t, next.Body.String(), `action="/admin/login"`)
	// Another account's session survives; the administrator can also use HTML logout.
	require.Equal(t, 200, taskRequest139(a, "GET", "/admin", a.cookie, nil, nil).Code)
	adminLogout := url.Values{"csrf_token": {taskField139(t, other, "csrf_token")}, "submission_token": {taskField139(t, other, "submission_token")}}
	require.Equal(t, 303, taskRequest139(a, "POST", "/admin/logout", a.cookie, adminLogout, nil).Code)
}

// TestUserManagementRoutesStayForbidden159 checks media reads and writes with a real ordinary session.
func TestUserManagementRoutesStayForbidden159(t *testing.T) {
	a := taskApp139(t)
	_, err := a.store.SaveUser(context.Background(), db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	session, err := admin.Login(context.Background(), a.store, "listener", "test-password", false)
	require.NoError(t, err)
	for _, path := range []string{"/second/manage/track/12", "/second/manage/track/12/edit", "/second/manage/track/12/review?action=delete", "/second/manage/artist/10", "/second/manage/album/11"} {
		require.Equal(t, 403, taskRequest139(a, "GET", path, session.ID.String(), nil, nil).Code, path)
		require.Equal(t, 200, taskRequest139(a, "GET", path, a.cookie, nil, nil).Code, path)
	}
	for _, path := range []string{"/second/manage/track/12/metadata", "/second/manage/track/12/confirm"} {
		require.Equal(t, 403, taskRequest139(a, "POST", path, session.ID.String(), url.Values{}, nil).Code, path)
	}
}
