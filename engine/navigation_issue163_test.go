package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/stretchr/testify/require"
)

// TestNavigationByPermission163 checks the raw HTML and direct management routes.
func TestNavigationByPermission163(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	other := ordinarySession161(t, a)
	require.NoError(t, a.store.Exec(fmt.Sprintf("INSERT INTO listens(profile_id,track_id,listened_at) VALUES(%d,12,100)", p.ID)))
	scope := "/" + p.Slug
	own := fmt.Sprintf("/admin?profile=%d", p.ID)
	account := "/account"
	for _, actor := range []struct {
		name, cookie  string
		listens, root bool
	}{
		{"visitor", "", false, false},
		{"ordinary third party", other, false, false},
		{"owner", owner, true, false},
		{"root", a.cookie, true, true},
	} {
		t.Run(actor.name, func(t *testing.T) {
			directory := taskRequest139(a, "GET", "/", actor.cookie, nil, nil)
			require.NotContains(t, directory.Body.String(), "Source Track")
			require.Equal(t, actor.root, strings.Contains(directory.Body.String(), `href="/admin/settings"`))
			require.Equal(t, actor.cookie != "", strings.Contains(directory.Body.String(), account))
			page := taskRequest139(a, "GET", scope+"/listens", actor.cookie, nil, nil)
			require.Equal(t, 200, page.Code)
			require.Equal(t, actor.listens, strings.Contains(page.Body.String(), scope+"/manage/track/12/listen?"))
			media := taskRequest139(a, "GET", scope+"/track/12", actor.cookie, nil, nil)
			require.Equal(t, 200, media.Code)
			require.Equal(t, actor.root, strings.Contains(media.Body.String(), "data-artwork-drop"))
			require.Equal(t, actor.root, strings.Contains(media.Body.String(), scope+"/manage/track/12/edit"))
			head := taskRequest139(a, "HEAD", scope+"/listens", actor.cookie, nil, nil)
			require.Equal(t, page.Code, head.Code)
			require.Empty(t, head.Body.String())
			require.Equal(t, page.Header().Get("Cache-Control"), head.Header().Get("Cache-Control"))
			require.Contains(t, head.Header().Values("Vary"), "Cookie")
		})
	}

	for _, path := range []string{"/admin", "/admin/settings", "/admin?profile=2", "/admin?section=account&profile=2", scope + "/manage/track/12/edit"} {
		denied := taskRequest139(a, "GET", path, owner, nil, nil)
		require.Equal(t, 403, denied.Code, path)
		for _, want := range []string{account, "Directory", "Sign out"} {
			require.Contains(t, denied.Body.String(), want, path)
		}
		for _, hidden := range []string{"/admin/settings", "Source Track", "account/password", "profile/create"} {
			require.NotContains(t, denied.Body.String(), hidden, path)
		}
	}
	page := taskRequest139(a, "GET", own, owner, nil, nil)
	require.Contains(t, page.Body.String(), "Only you and administrators can view protected listens.")
	require.Contains(t, page.Body.String(), `aria-describedby="protected-relay-help"`)
	root := taskRequest139(a, "GET", own, a.cookie, nil, nil)
	require.Contains(t, root.Body.String(), "Only this profile’s owner and administrators can view protected listens.")
	page = taskRequest139(a, "GET", account, owner, nil, nil)
	require.Regexp(t, `href="/account"\s+aria-current="page"`, page.Body.String())
	menu := taskRequest139(a, "GET", scope, owner, nil, nil)
	require.Contains(t, menu.Body.String(), `href="/admin/sign-out?return=`)
	require.NotContains(t, menu.Body.String(), `name="submission_token"`)
	confirm := taskRequest139(a, "GET", "/admin/sign-out", owner, nil, nil)
	form := url.Values{"csrf_token": {taskField139(t, confirm, "csrf_token")}, "submission_token": {taskField139(t, confirm, "submission_token")}}
	form.Set("username", "must-not-change")
	require.Equal(t, 409, taskRequest139(a, "POST", "/admin/account/username", owner, form, nil).Code)
	form.Del("username")
	require.Equal(t, 403, taskRequest139(a, "POST", "/admin/logout", other, form, nil).Code)
	require.Equal(t, 303, taskRequest139(a, "POST", "/admin/logout", owner, form, nil).Code)
	require.Equal(t, 401, taskRequest139(a, "POST", "/admin/logout", owner, form, nil).Code)
	require.Equal(t, 200, taskRequest139(a, "GET", account, other, nil, nil).Code)
}

// TestLoginReturn163 follows real POSTs and checks read authorization with the new session.
func TestLoginReturn163(t *testing.T) {
	a := taskApp139(t)
	_, p := owner160(t, a)
	ctx := context.Background()
	require.NoError(t, a.store.Exec(fmt.Sprintf("UPDATE listening_profiles SET require_login=1 WHERE id IN (2,%d)", p.ID)))
	scope := "/" + p.Slug
	const account = "/account"
	for _, tc := range []struct{ username, password, next, want string }{
		{"owner160", "test-password", "", account},
		{"owner160", "test-password", scope + "/listens", scope + "/listens"},
		{"owner160", "test-password", "/default/search?q=music", "/default/search?q=music"},
		{"owner160", "test-password", "/second/listens", account},
		{"owner160", "test-password", "/missing/listens", account},
		{"owner160", "test-password", "/default/track/999999", account},
		{"owner160", "test-password", scope + "/listens?month=1&week=1", account},
		{"owner160", "test-password", "/admin/settings", account},
		{"owner160", "test-password", "/second/manage/track/12/edit", account},
		{"owner160", "test-password", "//external.test/", account},
	} {
		t.Run(tc.next, func(t *testing.T) {
			page := taskRequest139(a, "GET", "/admin?next="+url.QueryEscape(tc.next), "", nil, nil)
			form := url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "username": {tc.username}, "password": {tc.password}, "next": {tc.next}}
			r := httptest.NewRequest("POST", "http://koito.test/admin/login", strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			for _, cookie := range page.Result().Cookies() {
				r.AddCookie(cookie)
			}
			w := httptest.NewRecorder()
			a.router.ServeHTTP(w, r)
			require.Equal(t, 303, w.Code)
			require.Equal(t, tc.want, w.Header().Get("Location"))
			for _, cookie := range w.Result().Cookies() {
				if cookie.Name == "tunetrail_session" {
					require.Equal(t, 200, taskRequest139(a, "GET", tc.want, cookie.Value, nil, nil).Code)
				}
			}
		})
	}
	// An expired session must not restore protected reads or account actions.
	session, err := admin.Login(ctx, a.store, "owner160", "test-password", false)
	require.NoError(t, err)
	require.NoError(t, a.store.Exec(fmt.Sprintf("UPDATE sessions SET expires_at=1 WHERE id='%s'", session.ID)))
	denied := taskRequest139(a, "GET", scope, session.ID.String(), nil, nil)
	require.Equal(t, 404, denied.Code)
	require.NotContains(t, denied.Body.String(), account)
	require.NotContains(t, denied.Body.String(), "Source Track")
}

// TestLegacyDocumentPrivacy163 compares both outcomes of the unscoped profile redirect.
func TestLegacyDocumentPrivacy163(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec("DELETE FROM listening_profiles WHERE id <> 1; UPDATE listening_profiles SET require_login=1 WHERE id=1"))
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		denied := taskRequest139(a, method, "/listens", "", nil, nil)
		missing := taskRequest139(a, method, "/missing/listens", "", nil, nil)
		require.Equal(t, 404, denied.Code)
		require.Equal(t, missing.Body.String(), denied.Body.String())
		require.Equal(t, missing.Header(), denied.Header())
		allowed := taskRequest139(a, method, "/listens", a.cookie, nil, nil)
		require.Equal(t, http.StatusFound, allowed.Code)
		require.Equal(t, "/default/listens", allowed.Header().Get("Location"))
		for _, w := range []*httptest.ResponseRecorder{allowed, denied} {
			require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
			require.Contains(t, w.Header().Values("Vary"), "Cookie")
		}
	}
}
