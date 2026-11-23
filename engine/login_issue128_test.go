package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// signIn128 loads the neutral sign-in document and posts the form the way a browser does,
// carrying the pre-session cookie. It returns the POST response.
func signIn128(t *testing.T, a *profileTestApp, username, password, next string) *httptest.ResponseRecorder {
	t.Helper()
	page := taskRequest139(a, "GET", "/login", "", nil, nil)
	require.Equal(t, 200, page.Code)
	var cookies []string
	for _, c := range page.Result().Cookies() {
		cookies = append(cookies, c.Name+"="+c.Value)
	}
	form := url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "username": {username}, "password": {password}, "next": {next}}
	return taskRequest139(a, "POST", "/admin/login", "", form, map[string]string{"Cookie": strings.Join(cookies, "; ")})
}

// users128 creates an ordinary user and returns its session cookie; root keeps a.cookie.
func users128(t *testing.T, a *profileTestApp, name string) string {
	t.Helper()
	ctx := context.Background()
	_, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: name, Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	s, err := admin.Login(ctx, a.store, name, "test-password", false)
	require.NoError(t, err)
	return s.ID.String()
}

// TestLoginRouteEveryRole128 signs in at /login as an ordinary user and as root, and checks the
// safe return: unknown, external, looping and unauthorized destinations fall back to the role's home.
func TestLoginRouteEveryRole128(t *testing.T) {
	a := taskApp139(t)
	users128(t, a, "listener")
	visitor := taskRequest139(a, "GET", "/login", "", nil, nil)
	body := visitor.Body.String()
	require.Contains(t, body, `class="account-card auth-card"`)
	require.Contains(t, body, `action="/admin/login"`)
	require.Contains(t, body, `name="next" value=""`)
	// A plain visit has nothing to explain; a protected document asks the visitor to sign in.
	require.NotContains(t, body, `class="message `)
	protected := taskRequest139(a, "GET", "/admin", "", nil, nil).Body.String()
	require.Contains(t, protected, `class="message message-warning" role="status"`)
	require.Contains(t, protected, "Sign in to continue.")
	for _, tc := range []struct{ name, user, next, want string }{
		{"ordinary home", "listener", "", "/account"},
		{"ordinary returns to a tab", "listener", "/account/keys", "/account/keys"},
		{"ordinary public read", "listener", "/second", "/second"},
		{"ordinary denied root page", "listener", "/admin/settings", "/account"},
		{"ordinary denied other profile", "listener", "/admin?profile=2", "/account"},
		{"external", "listener", "https://evil.test/account", "/account"},
		{"scheme-relative", "listener", "//evil.test", "/account"},
		{"backslash", "listener", `/\evil.test`, "/account"},
		{"unknown path", "listener", "/nowhere/at/all/x", "/account"},
		{"login loop", "listener", "/login", "/account"},
		{"root home", "administrator", "", "/admin"},
		{"root settings", "administrator", "/admin/settings", "/admin/settings"},
		{"root external", "administrator", "https://evil.test", "/admin"},
	} {
		// The sign-in limiter allows ten attempts a minute per app, so every case gets its own.
		a := taskApp139(t)
		users128(t, a, "listener")
		out := signIn128(t, a, tc.user, "test-password", tc.next)
		require.Equal(t, 303, out.Code, tc.name)
		require.Equal(t, tc.want, out.Header().Get("Location"), tc.name)
	}
}

// TestLoginRouteSignedIn128 sends an open session straight to its validated destination.
func TestLoginRouteSignedIn128(t *testing.T) {
	a := taskApp139(t)
	user := users128(t, a, "listener")
	for _, tc := range []struct{ cookie, path, want string }{
		{user, "/login", "/account"},
		{user, "/login?next=%2Faccount%2Fkeys", "/account/keys"},
		{user, "/login?next=%2Fadmin%2Fsettings", "/account"},
		{user, "/login?next=https%3A%2F%2Fevil.test", "/account"},
		{a.cookie, "/login", "/admin"},
		{a.cookie, "/login?next=%2Fadmin%2Fsettings", "/admin/settings"},
	} {
		out := taskRequest139(a, "GET", tc.path, tc.cookie, nil, nil)
		require.Equal(t, 303, out.Code, tc.path)
		require.Equal(t, tc.want, out.Header().Get("Location"), tc.path)
	}
	// The legacy entry keeps working for visitors and POST contracts are unchanged.
	require.Contains(t, taskRequest139(a, "GET", "/admin", "", nil, nil).Body.String(), `action="/admin/login"`)
	require.Equal(t, 405, taskRequest139(a, "POST", "/login", user, url.Values{}, nil).Code)
}

// TestLoginExpiredSession128 shows the neutral instruction for a stale cookie, in both languages.
func TestLoginExpiredSession128(t *testing.T) {
	a := taskApp139(t)
	user := users128(t, a, "listener")
	require.NoError(t, a.store.Exec("UPDATE sessions SET expires_at=1 WHERE id=?", user))
	for locale, want := range map[string]string{"en": "Sign in to continue.", "pt-BR": "Entre para continuar."} {
		cookie := "tunetrail_session=" + user + "; tunetrail_language=" + locale
		for _, path := range []string{"/login", "/account", "/admin"} {
			out := taskRequest139(a, "GET", path, "", nil, map[string]string{"Cookie": cookie})
			require.Equal(t, 200, out.Code, path)
			require.Contains(t, out.Body.String(), want, locale+path)
			require.Contains(t, out.Body.String(), `class="message message-warning" role="status"`, locale+path)
			require.Contains(t, out.Body.String(), `action="/admin/login"`, locale+path)
			require.NotContains(t, out.Body.String(), `class="message message-error"`, locale+path)
		}
	}
	// The protected page is the return target after signing in again.
	out := taskRequest139(a, "GET", "/account/keys", "", nil, map[string]string{"Cookie": "tunetrail_session=" + user})
	require.Contains(t, out.Body.String(), `name="next" value="/account/keys"`)
}

// TestLoginInvalidCredentialsUniform128 gives unknown users, wrong passwords and pending accounts
// the same status and message.
func TestLoginInvalidCredentialsUniform128(t *testing.T) {
	a := taskApp139(t)
	users128(t, a, "listener")
	users128(t, a, "pending")
	require.NoError(t, a.store.Exec("UPDATE users SET status='pending',password='' WHERE username='pending'"))
	normalize := regexp.MustCompile(`name="csrf_token"\s+value="[^"]*"`)
	var first string
	for _, tc := range []struct{ user, password string }{
		{"listener", "wrong-password"}, {"nobody", "test-password"}, {"pending", "test-password"}, {"administrator", "wrong-password"},
	} {
		out := signIn128(t, a, tc.user, tc.password, "")
		require.Equal(t, 401, out.Code, tc.user)
		body := normalize.ReplaceAllString(out.Body.String(), "")
		require.Contains(t, body, `class="message message-error" role="alert"`, tc.user)
		require.Contains(t, body, "Invalid username or password.", tc.user)
		require.Contains(t, body, `aria-describedby="form-error"`, tc.user)
		if first == "" {
			first = body
		}
		require.Equal(t, first, body, tc.user)
	}
}

// TestNeutralRefusals128 checks the 401/403/404 documents: fixed messages, session-coherent entries,
// and no way to tell a hidden profile from a missing one.
func TestNeutralRefusals128(t *testing.T) {
	a := taskApp139(t)
	user := users128(t, a, "listener")
	require.NoError(t, a.store.Exec("UPDATE listening_profiles SET require_login=1 WHERE id=2"))

	// 401: a visitor is asked to sign in; no account entries.
	out := taskRequest139(a, "GET", "/admin/profile/2/export", "", nil, nil)
	require.Equal(t, 401, out.Code)
	body := out.Body.String()
	for _, want := range []string{"Sign in required", "Sign in to continue.", `href="/login"`, `class="message message-warning" role="status"`} {
		require.Contains(t, body, want)
	}
	require.NotContains(t, body, "/admin/sign-out")

	// 403: a signed-in ordinary user keeps its own entries and is not told to sign in.
	forbidden := taskRequest139(a, "GET", "/admin", user, nil, nil)
	require.Equal(t, 403, forbidden.Code)
	for _, want := range []string{"Access denied", "Your account does not have permission to access this page.", `class="message message-error" role="alert"`, `href="/account"`, `href="/admin/sign-out`} {
		require.Contains(t, forbidden.Body.String(), want)
	}
	require.NotContains(t, forbidden.Body.String(), `href="/login"`)
	pt := taskRequest139(a, "GET", "/admin", "", nil, map[string]string{"Cookie": "tunetrail_session=" + user + "; tunetrail_language=pt-BR"})
	require.Contains(t, pt.Body.String(), "Acesso negado")
	require.Contains(t, pt.Body.String(), "Sua conta não tem permissão para acessar esta página.")

	// 404: a protected profile and a missing one answer byte for byte the same, for visitor and user.
	for _, cookie := range []string{"", user} {
		hidden := taskRequest139(a, "GET", "/admin/profile/2/export", cookie, nil, nil)
		missing := taskRequest139(a, "GET", "/admin/profile/999999/export", cookie, nil, nil)
		if cookie == "" {
			// Without a session both are the same 401, never a 404 that could differ.
			require.Equal(t, 401, hidden.Code)
			require.Equal(t, hidden.Code, missing.Code)
			require.Equal(t, hidden.Body.String(), missing.Body.String())
			continue
		}
		require.Equal(t, 404, hidden.Code)
		require.Equal(t, hidden.Code, missing.Code)
		require.Equal(t, hidden.Body.String(), missing.Body.String())
		require.Contains(t, hidden.Body.String(), "Page unavailable")
		require.Contains(t, hidden.Body.String(), "This profile or page is unavailable.")
		require.Contains(t, hidden.Body.String(), `href="/account"`)
		require.NotContains(t, hidden.Body.String(), "permission")
	}
	// The 403 and the 404 never share a title or message.
	require.NotContains(t, forbidden.Body.String(), "Page unavailable")
}

// TestPasswordChangeSignInNotice128 explains the sign-out that follows a password change.
func TestPasswordChangeSignInNotice128(t *testing.T) {
	a := taskApp139(t)
	cookie := users128(t, a, "listener")
	out := accountPost125(t, a, cookie, "password", url.Values{"password": {"new-password-1"}, "confirm_password": {"new-password-1"}}, nil)
	require.Equal(t, http.StatusSeeOther, out.Code)
	require.Equal(t, "/login?changed=1", out.Header().Get("Location"))
	for locale, want := range map[string]string{"en": "Password changed. Sign in with your new password.", "pt-BR": "Senha alterada. Entre com a nova senha."} {
		page := taskRequest139(a, "GET", "/login?changed=1", "", nil, map[string]string{"Cookie": "tunetrail_language=" + locale})
		require.Equal(t, 200, page.Code)
		require.Contains(t, page.Body.String(), want)
		require.Contains(t, page.Body.String(), `class="message message-success" role="status"`)
		require.NotContains(t, page.Body.String(), "Sign in to continue.")
	}
	// The new password signs in; the old one does not.
	require.Equal(t, 303, signIn128(t, a, "listener", "new-password-1", "").Code)
	require.Equal(t, 401, signIn128(t, a, "listener", "test-password", "").Code)
}

// TestLoginSlugReserved128 keeps /login from being claimed by a listening profile.
func TestLoginSlugReserved128(t *testing.T) {
	a := taskApp139(t)
	form := taskRequest139(a, "GET", "/admin", a.cookie, nil, nil)
	post := url.Values{"csrf_token": {taskField139(t, form, "csrf_token")}, "submission_token": {taskField139(t, form, "submission_token")}, "display_name": {"Login"}, "slug": {"login"}}
	out := taskRequest139(a, "POST", "/admin/profile/create", a.cookie, post, nil)
	require.GreaterOrEqual(t, out.Code, 400)
	profiles, err := a.store.ListListeningProfiles(context.Background())
	require.NoError(t, err)
	for _, p := range profiles {
		require.NotEqual(t, "login", p.Slug)
	}
}

// TestLoginLanguageSwitchStays128 keeps the user on the sign-in card when changing the language,
// and answers HEAD like GET.
func TestLoginLanguageSwitchStays128(t *testing.T) {
	a := taskApp139(t)
	page := taskRequest139(a, "GET", "/login?next=%2Faccount%2Fkeys", "", nil, nil)
	require.Contains(t, page.Body.String(), `name="return_to" value="/login?next=%2Faccount%2Fkeys"`)
	form := url.Values{"language": {"pt-BR"}, "return_to": {"/login?next=%2Faccount%2Fkeys"}}
	out := taskRequest139(a, "POST", "/language", "", form, nil)
	require.Equal(t, 303, out.Code)
	require.Equal(t, "/login?next=%2Faccount%2Fkeys", out.Header().Get("Location"))
	bad := taskRequest139(a, "POST", "/language", "", url.Values{"language": {"en"}, "return_to": {"/login?next=https%3A%2F%2Fevil.test"}}, nil)
	require.Equal(t, 400, bad.Code)
	head := taskRequest139(a, "HEAD", "/login", "", nil, nil)
	require.Equal(t, 200, head.Code)
	require.Empty(t, head.Body.String())
}
