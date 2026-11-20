package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const signOut157 = "/admin/sign-out"

func signOutForm157(t *testing.T, page *httptest.ResponseRecorder) url.Values {
	t.Helper()
	require.Equal(t, http.StatusOK, page.Code)
	require.Contains(t, page.Body.String(), `action="/admin/logout"`)
	return url.Values{
		"csrf_token":       {taskField139(t, page, "csrf_token")},
		"submission_token": {taskField139(t, page, "submission_token")},
	}
}

// TestSignOutConfirmation157 checks read-only confirmation, localization and authorized cancellation.
func TestSignOutConfirmation157(t *testing.T) {
	a := taskApp139(t)
	ordinary := ordinarySession161(t, a)
	for _, tc := range []struct {
		cookie, locale, title, label, cancel string
	}{
		{ordinary, "pt-BR", "Encerrar esta sessão neste navegador", "Sair", "Cancelar"},
		{a.cookie, "en", "End this session in this browser", "Sign out", "Cancel"},
	} {
		headers := map[string]string{"Cookie": adminweb.SessionCookie + "=" + tc.cookie + "; tunetrail_language=" + tc.locale}
		page := taskRequest139(a, "GET", signOut157+"?return=%2F", "", nil, headers)
		signOutForm157(t, page)
		require.Equal(t, "private, no-store", page.Header().Get("Cache-Control"))
		require.Contains(t, page.Header().Values("Vary"), "Cookie")
		require.Contains(t, page.Body.String(), tc.title)
		require.Contains(t, page.Body.String(), `type="submit" aria-describedby="sign-out-description">`+tc.label+`</button>`)
		require.Contains(t, page.Body.String(), `href="/" autofocus aria-describedby="sign-out-description">`+tc.cancel+`</a>`)
		head := taskRequest139(a, "HEAD", signOut157, tc.cookie, nil, nil)
		require.Equal(t, http.StatusOK, head.Code)
		require.Equal(t, page.Header().Get("Cache-Control"), head.Header().Get("Cache-Control"))
		require.Empty(t, head.Body.String())
		prefetch := taskRequest139(a, "GET", signOut157, tc.cookie, nil, map[string]string{"Sec-Purpose": "prefetch;prerender"})
		require.Equal(t, http.StatusNoContent, prefetch.Code)
		require.Empty(t, prefetch.Body.String())
		for _, bad := range []string{"//outside.test", "https://outside.test/", "javascript:alert(1)", "/admin/logout"} {
			cancel := taskRequest139(a, "GET", signOut157+"?return="+url.QueryEscape(bad), tc.cookie, nil, nil)
			require.Contains(t, cancel.Body.String(), `href="/admin?section=account" autofocus aria-describedby="sign-out-description"`)
		}
		_, err := admin.Authenticate(context.Background(), a.store, uuid.MustParse(tc.cookie))
		require.NoError(t, err)
	}
	denied := taskRequest139(a, "GET", signOut157+"?return=%2Fadmin%2Fsettings", ordinary, nil, nil)
	require.Contains(t, denied.Body.String(), `href="/admin?section=account" autofocus aria-describedby="sign-out-description"`)
	allowed := taskRequest139(a, "GET", signOut157+"?return=%2Fadmin%2Fsettings", a.cookie, nil, nil)
	require.Contains(t, allowed.Body.String(), `href="/admin/settings" autofocus aria-describedby="sign-out-description"`)
	missing := taskRequest139(a, "GET", signOut157, "", nil, nil)
	require.Equal(t, http.StatusUnauthorized, missing.Code)
	require.Contains(t, missing.Body.String(), "Sign in to continue.")
	require.Contains(t, missing.Body.String(), `action="/admin/login"`)
}

// TestSignOutSubmission157 keeps the ticket, CSRF and revocation scoped to one session.
func TestSignOutSubmission157(t *testing.T) {
	a := taskApp139(t)
	ordinary := ordinarySession161(t, a)
	post := func(cookie string, f url.Values) *httptest.ResponseRecorder {
		return taskRequest139(a, "POST", "/admin/logout", cookie, f, nil)
	}
	rootForm := signOutForm157(t, taskRequest139(a, "GET", signOut157, a.cookie, nil, nil))
	secondTab := signOutForm157(t, taskRequest139(a, "GET", signOut157, a.cookie, nil, nil))
	ordinaryForm := signOutForm157(t, taskRequest139(a, "GET", signOut157, ordinary, nil, nil))
	edit := taskRequest139(a, "GET", "/admin?section=account", a.cookie, nil, nil)
	for _, value := range []string{"", "invalid", ordinaryForm.Get("csrf_token")} {
		f := url.Values{"submission_token": {rootForm.Get("submission_token")}}
		if value != "" {
			f.Set("csrf_token", value)
		}
		require.Equal(t, http.StatusForbidden, post(a.cookie, f).Code)
	}
	for _, ticket := range []string{"", ordinaryForm.Get("submission_token"), taskField139(t, edit, "submission_token")} {
		f := url.Values{"csrf_token": {rootForm.Get("csrf_token")}, "submission_token": {ticket}}
		require.Equal(t, http.StatusConflict, post(a.cookie, f).Code)
	}

	// Ambiguous cookies in one browser are rejected before any session is selected.
	ambiguous := taskRequest139(a, "POST", "/admin/logout", "", rootForm, map[string]string{
		"Cookie": adminweb.SessionCookie + "=" + a.cookie + "; " + adminweb.SessionCookie + "=" + ordinary,
	})
	require.Equal(t, http.StatusUnauthorized, ambiguous.Code)
	require.Contains(t, ambiguous.Body.String(), `action="/admin/login"`)

	closed := taskRequest139(a, "POST", "/admin/logout", "", secondTab, map[string]string{
		"Cookie": adminweb.SessionCookie + "=" + a.cookie + "; tunetrail_preauth=obsolete",
	})
	require.Equal(t, http.StatusSeeOther, closed.Code)
	require.Equal(t, "/", closed.Header().Get("Location"))
	for _, c := range closed.Result().Cookies() {
		require.Empty(t, c.Value)
		require.Less(t, c.MaxAge, 0)
	}
	require.Len(t, closed.Result().Cookies(), 2)
	stale := post(a.cookie, rootForm)
	require.Equal(t, http.StatusUnauthorized, stale.Code)
	require.Contains(t, stale.Body.String(), "Sign in to continue.")
	require.Contains(t, stale.Body.String(), `action="/admin/login"`)
	directory := taskRequest139(a, "GET", closed.Header().Get("Location"), "", nil, nil)
	require.Contains(t, directory.Body.String(), "Sign in")
	require.NotContains(t, directory.Body.String(), "/admin/sign-out")
	_, err := admin.Authenticate(context.Background(), a.store, uuid.MustParse(ordinary))
	require.NoError(t, err)

	require.NoError(t, a.store.Exec("UPDATE sessions SET expires_at=1 WHERE id=?", ordinary))
	expired := taskRequest139(a, "POST", "/admin/logout", "", ordinaryForm, map[string]string{
		"Cookie": adminweb.SessionCookie + "=" + ordinary + "; tunetrail_language=pt-BR",
	})
	require.Equal(t, http.StatusUnauthorized, expired.Code)
	require.Contains(t, expired.Body.String(), "Entre para continuar.")
	require.Contains(t, expired.Body.String(), `action="/admin/login"`)
}

// TestSignOutLinksKeepReadOrigin157 follows the return offered by each page.
func TestSignOutLinksKeepReadOrigin157(t *testing.T) {
	a := taskApp139(t)
	ordinary := ordinarySession161(t, a)
	for _, tc := range []struct{ path, cookie string }{
		{"/admin/settings", a.cookie},
		{"/admin?section=profiles", a.cookie},
		{"/admin?section=account", ordinary},
		{"/second/manage/track/12/edit", a.cookie},
		{"/", ordinary},
	} {
		page := taskRequest139(a, "GET", tc.path, tc.cookie, nil, nil)
		require.Equal(t, http.StatusOK, page.Code, tc.path)
		link := signOut157 + "?return=" + url.QueryEscape(tc.path)
		require.Contains(t, page.Body.String(), `href="`+link+`"`)
		confirm := taskRequest139(a, "GET", link, tc.cookie, nil, nil)
		require.Equal(t, http.StatusOK, confirm.Code)
		require.Contains(t, confirm.Body.String(), `href="`+tc.path+`" autofocus aria-describedby="sign-out-description"`)
	}

	invalid := taskRequest139(a, "GET", "/admin?section=account&next=https://outside.test", ordinary, nil, nil)
	require.Equal(t, http.StatusOK, invalid.Code)
	require.Contains(t, invalid.Body.String(), `href="/admin/sign-out"`)
	require.NotContains(t, invalid.Body.String(), "/admin/sign-out?return=")
}
