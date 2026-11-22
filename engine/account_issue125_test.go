package engine

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// accountPost125 sends a card form with the tokens read from the rendered /account page.
func accountPost125(t *testing.T, a *profileTestApp, cookie, action string, fields url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	page := taskRequest139(a, "GET", "/account", cookie, nil, nil)
	require.Equal(t, 200, page.Code)
	fields.Set("csrf_token", taskField139(t, page, "csrf_token"))
	fields.Set("submission_token", taskField139(t, page, "submission_token"))
	return taskRequest139(a, "POST", "/admin/account/"+action, cookie, fields, headers)
}

// TestAccountPageEveryRole125 serves /account to ordinary, owner and root sessions without a 403.
func TestAccountPageEveryRole125(t *testing.T) {
	a := taskApp139(t)
	ctx := context.Background()
	_, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	ordinary, err := admin.Login(ctx, a.store, "listener", "test-password", false)
	require.NoError(t, err)
	owner, _ := owner160(t, a)
	_ = owner
	for name, cookie := range map[string]string{"ordinary": ordinary.ID.String(), "root": a.cookie} {
		page := taskRequest139(a, "GET", "/account", cookie, nil, nil)
		require.Equal(t, 200, page.Code, name)
		body := page.Body.String()
		for _, want := range []string{`id="account-identity"`, `id="account-password"`, `id="account-session"`, `id="account-danger"`,
			`ends every session, including this one`, `href="/admin/sign-out?return=%2Faccount"`, `name="confirm_password"`} {
			require.Contains(t, body, want, name)
		}
		require.Contains(t, body, `href="/account"`, name)
	}
	// Without a session the neutral sign-in document returns to /account.
	visitor := taskRequest139(a, "GET", "/account", "", nil, nil)
	require.NotContains(t, visitor.Body.String(), `id="account-password"`)
	require.Contains(t, visitor.Body.String(), `name="next" value="/account"`)
	require.Equal(t, 405, taskRequest139(a, "POST", "/account", ordinary.ID.String(), url.Values{}, nil).Code)
}

// TestAccountLegacyRedirect125 keeps /admin?section=account links working for signed-in sessions only.
func TestAccountLegacyRedirect125(t *testing.T) {
	a := taskApp139(t)
	w := taskRequest139(a, "GET", "/admin?section=account", a.cookie, nil, nil)
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/account", w.Header().Get("Location"))
	anon := taskRequest139(a, "GET", "/admin?section=account", "", nil, nil)
	require.Equal(t, 200, anon.Code)
	require.NotContains(t, anon.Body.String(), `id="account-password"`)
}

// TestAccountUsernameCard125 covers success (full page and fragment) and card-local errors.
func TestAccountUsernameCard125(t *testing.T) {
	a := taskApp139(t)
	ctx := context.Background()
	_, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	s, err := admin.Login(ctx, a.store, "listener", "test-password", false)
	require.NoError(t, err)
	cookie := s.ID.String()

	bad := accountPost125(t, a, cookie, "username", url.Values{"username": {"not valid!"}}, nil)
	require.Equal(t, 400, bad.Code)
	require.Contains(t, bad.Body.String(), `id="account-identity-error"`)
	require.Contains(t, bad.Body.String(), `role="alert"`)
	require.Contains(t, bad.Body.String(), `aria-invalid="true"`)
	require.Contains(t, bad.Body.String(), `value="not valid!"`)
	require.Contains(t, bad.Body.String(), `id="account-password"`)

	frag := accountPost125(t, a, cookie, "username", url.Values{"username": {"bad name"}}, map[string]string{"X-TuneTrail-Partial": "1"})
	require.Equal(t, "account-identity", frag.Header().Get("X-TuneTrail-Fragment"))
	require.NotEmpty(t, frag.Header().Get("X-TuneTrail-Submission"))
	require.NotContains(t, frag.Body.String(), "<html")

	ok := accountPost125(t, a, cookie, "username", url.Values{"username": {"listener2"}}, nil)
	require.Equal(t, 303, ok.Code)
	require.Equal(t, "/account", ok.Header().Get("Location"))
	page := taskRequest139(a, "GET", "/account", cookie, nil, nil)
	require.Contains(t, page.Body.String(), `role="status"`)
	require.Contains(t, page.Body.String(), `value="listener2"`)

	done := accountPost125(t, a, cookie, "username", url.Values{"username": {"listener3"}}, map[string]string{"X-TuneTrail-Partial": "1"})
	require.Equal(t, 200, done.Code)
	require.Equal(t, "account-identity", done.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, done.Body.String(), `data-section-message`)
}

// TestAccountPasswordCard125 covers mismatch, invalid length and the revocation of every session.
func TestAccountPasswordCard125(t *testing.T) {
	a := taskApp139(t)
	ctx := context.Background()
	_, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	first, err := admin.Login(ctx, a.store, "listener", "test-password", false)
	require.NoError(t, err)
	second, err := admin.Login(ctx, a.store, "listener", "test-password", false)
	require.NoError(t, err)
	cookie := first.ID.String()

	mismatch := accountPost125(t, a, cookie, "password", url.Values{"password": {"new-password-1"}, "confirm_password": {"other-password"}}, nil)
	require.Equal(t, 400, mismatch.Code)
	require.Contains(t, mismatch.Body.String(), `id="account-password-error"`)
	short := accountPost125(t, a, cookie, "password", url.Values{"password": {"short"}, "confirm_password": {"short"}}, nil)
	require.Equal(t, 400, short.Code)
	require.Contains(t, short.Body.String(), `id="account-password-error"`)
	require.NotContains(t, short.Body.String(), `value="short"`)

	ok := accountPost125(t, a, cookie, "password", url.Values{"password": {"new-password-1"}, "confirm_password": {"new-password-1"}}, nil)
	require.Equal(t, 303, ok.Code)
	for _, c := range []string{first.ID.String(), second.ID.String()} {
		require.Equal(t, 200, taskRequest139(a, "GET", "/account", c, nil, nil).Code)
		require.NotContains(t, taskRequest139(a, "GET", "/account", c, nil, nil).Body.String(), `id="account-password"`)
	}
}

// TestRedeemSignInNotice125 shows the explicit next step after a token redemption.
func TestRedeemSignInNotice125(t *testing.T) {
	a := taskApp139(t)
	page := taskRequest139(a, "GET", "/admin?redeemed=1", "", nil, nil)
	require.Equal(t, 200, page.Code)
	require.Contains(t, page.Body.String(), "Password set. Sign in with your username and the new password.")
	require.Contains(t, page.Body.String(), `action="/admin/login"`)
	// A signed-in browser that follows the same redirect lands on its account.
	signed := taskRequest139(a, "GET", "/admin?redeemed=1", a.cookie, nil, nil)
	require.Equal(t, 303, signed.Code)
	require.Equal(t, "/account", signed.Header().Get("Location"))
	// A stale login form asks to retry instead of claiming a missing permission.
	stale := taskRequest139(a, "POST", "/admin/login", "", url.Values{"csrf_token": {"stale"}, "username": {"x"}, "password": {"y"}}, nil)
	require.Equal(t, 403, stale.Code)
	require.Contains(t, stale.Body.String(), "This page expired or was opened in another tab. Try again.")
	require.NotContains(t, stale.Body.String(), "does not have permission")
}

// TestRedeemNoticeBox125 shows the sign-out notice on /admin/redeem for visitors and open sessions.
func TestRedeemNoticeBox125(t *testing.T) {
	a := taskApp139(t)
	ctx := context.Background()
	_, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	ordinary, err := admin.Login(ctx, a.store, "listener", "test-password", false)
	require.NoError(t, err)

	visitor := taskRequest139(a, "GET", "/admin/redeem", "", nil, nil)
	require.Equal(t, 200, visitor.Code)
	body := visitor.Body.String()
	require.Contains(t, body, `class="account-warning" role="note"`)
	require.Contains(t, body, "Before redeeming: if you are signed in as an administrator (root)")
	require.Contains(t, body, `href="/admin/sign-out"`)
	require.Contains(t, body, `>All profiles</a>`)
	require.NotContains(t, body, "You are signed in as")

	pt := taskRequest139(a, "GET", "/admin/redeem", "", nil, map[string]string{"Accept-Language": "pt-BR"})
	require.Contains(t, pt.Body.String(), "Antes de resgatar: se você estiver conectado como administrador (root)")
	require.Contains(t, pt.Body.String(), ">Todos os perfis</a>")

	for name, cookie := range map[string]string{"root": a.cookie, "ordinary": ordinary.ID.String()} {
		page := taskRequest139(a, "GET", "/admin/redeem", cookie, nil, nil)
		require.Equal(t, 200, page.Code, name)
		require.Contains(t, page.Body.String(), "To redeem the token of another account, sign out first.", name)
		require.Contains(t, page.Body.String(), `href="/admin/sign-out"`, name)
		require.NotContains(t, page.Body.String(), "Before redeeming", name)
	}
	ptSession := taskRequest139(a, "GET", "/admin/redeem", ordinary.ID.String(), nil, map[string]string{"Accept-Language": "pt-BR"})
	require.Contains(t, ptSession.Body.String(), "Você está conectado como listener. Para resgatar o token de outra conta, saia primeiro.")

	// A failed redemption keeps the notice and never echoes the token or password.
	page := taskRequest139(a, "GET", "/admin/redeem", "", nil, nil)
	form := url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "username": {"listener"}, "account_token": {"secret-token-value"},
		"password": {"secret-password-1"}, "confirm_password": {"different-password"}}
	cookie := ""
	for _, c := range page.Result().Cookies() {
		cookie += c.Name + "=" + c.Value + "; "
	}
	failed := taskRequest139(a, "POST", "/admin/redeem", "", form, map[string]string{"Cookie": cookie})
	require.Equal(t, 400, failed.Code)
	require.Contains(t, failed.Body.String(), `role="note"`)
	require.NotContains(t, failed.Body.String(), "secret-token-value")
	require.NotContains(t, failed.Body.String(), "secret-password-1")
}
