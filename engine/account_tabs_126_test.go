package engine

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// rootOnly126 lists markers of controls that only an administrator may ever receive.
var rootOnly126 = []string{"account-token", "account-role", "delete-review", "/admin/profile/create", `href="/admin/settings`,
	`href="/admin?section=profiles"`, "account-rows"}

// profileOf126 returns the profile ID owned by a username.
func profileOf126(t *testing.T, a *profileTestApp, username string) int32 {
	t.Helper()
	var id int32
	require.NoError(t, a.store.QueryRow(`SELECT lp.id FROM listening_profiles lp JOIN users u ON u.id=lp.user_id WHERE u.username=?`, username).Scan(&id))
	return id
}

// TestAccountTabsOwnAddress126 serves the five tabs from the session alone: no ID in the URL,
// no root control for the ordinary user or the owner, the same tabs for the root's own profile.
func TestAccountTabsOwnAddress126(t *testing.T) {
	a := taskApp139(t)
	owner, _ := owner160(t, a)
	ordinary := ordinarySession161(t, a)
	tabs := []string{"/account", "/account/profile", "/account/keys", "/account/history", "/account/relay"}
	for name, cookie := range map[string]string{"ordinary": ordinary, "owner": owner, "root": a.cookie} {
		for _, path := range tabs {
			w := taskRequest139(a, "GET", path, cookie, nil, nil)
			require.Equal(t, 200, w.Code, name+" "+path)
			body := w.Body.String()
			for _, tab := range tabs {
				require.Contains(t, body, `href="`+tab+`"`, name+" "+path)
			}
			require.NotContains(t, body, "you are operating as administrator", name+" "+path)
			require.NotContains(t, body, `class="account-tabs"`+"\n    <a href=\"/admin", name)
			// Health stays in the main menu, never in the tab strip.
			require.Contains(t, body, "/admin/health?profile=", name+" "+path)
			require.NotRegexp(t, `(?s)class="account-tabs".*?Health.*?</nav>`, body, name)
			if name != "root" {
				for _, hidden := range rootOnly126 {
					require.NotContains(t, body, hidden, name+" "+path)
				}
			}
			// Own-profile tabs never accept a target or any other query.
			require.Equal(t, 400, taskRequest139(a, "GET", path+"?profile=1", cookie, nil, nil).Code, path)
		}
		require.Contains(t, taskRequest139(a, "GET", "/account/profile", cookie, nil, nil).Body.String(), `name="display_name"`)
	}
	// Even for the root, the own tabs carry no root tool of another account.
	for _, path := range tabs {
		body := taskRequest139(a, "GET", path, a.cookie, nil, nil).Body.String()
		for _, hidden := range []string{"account-token", "account-role", "delete-review"} {
			require.NotContains(t, body, hidden, path)
		}
	}
	// The visitor gets the sign-in document and no tab content.
	for _, path := range tabs {
		w := taskRequest139(a, "GET", path, "", nil, nil)
		require.NotContains(t, w.Body.String(), `name="display_name"`, path)
		require.Contains(t, w.Body.String(), `name="username"`, path)
	}
	// There is no "My profile" menu entry any more.
	require.NotContains(t, taskRequest139(a, "GET", "/account", owner, nil, nil).Body.String(), "My profile")
}

// TestOwnProfileRedirects126 keeps every legacy URL of the own profile working through one address.
func TestOwnProfileRedirects126(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	for _, tc := range []struct{ query, want string }{
		{fmt.Sprintf("profile=%d", p.ID), "/account/profile"},
		{fmt.Sprintf("section=profiles&profile=%d", p.ID), "/account/profile"},
		{fmt.Sprintf("section=keys&profile=%d", p.ID), "/account/keys"},
		{fmt.Sprintf("section=import&profile=%d", p.ID), "/account/history"},
		{fmt.Sprintf("section=export&profile=%d", p.ID), "/account/history"},
		{fmt.Sprintf("section=relay&profile=%d", p.ID), "/account/relay"},
		{fmt.Sprintf("section=account&profile=%d", p.ID), "/account"},
		{"section=account", "/account"},
	} {
		w := taskRequest139(a, "GET", "/admin?"+tc.query, owner, nil, nil)
		require.Equal(t, 303, w.Code, tc.query)
		require.Equal(t, tc.want, w.Header().Get("Location"), tc.query)
	}
	// The root's own profile follows the same rule; another account does not redirect.
	w := taskRequest139(a, "GET", "/admin?profile=1", a.cookie, nil, nil)
	require.Equal(t, 303, w.Code)
	require.Equal(t, "/account/profile", w.Header().Get("Location"))
	require.Equal(t, 200, taskRequest139(a, "GET", fmt.Sprintf("/admin?profile=%d", p.ID), a.cookie, nil, nil).Code)
	// Nothing else about the old routes becomes reachable for the owner.
	for _, path := range []string{"/admin", "/admin?section=profiles", "/admin?profile=1", "/admin?section=account&profile=1", "/admin/settings"} {
		require.Equal(t, 403, taskRequest139(a, "GET", path, owner, nil, nil).Code, path)
	}
}

// TestRootContext126 shows another account in the administration context, with root tools that
// neither the owner nor an ordinary user can see or open.
func TestRootContext126(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	ordinary := ordinarySession161(t, a)
	otherID := profileOf126(t, a, "other161")
	base := fmt.Sprintf("/admin?profile=%d", p.ID)

	page := taskRequest139(a, "GET", base, a.cookie, nil, nil)
	require.Equal(t, 200, page.Code)
	body := page.Body.String()
	require.Contains(t, body, fmt.Sprintf("(@%s)", p.Slug))
	require.Contains(t, body, "you are operating as administrator")
	require.Contains(t, body, "You: ")
	require.Contains(t, body, `href="/admin?section=profiles"`)
	for _, tab := range []string{"/admin?section=account&amp;profile=%d", "/admin?profile=%d", "/admin?section=keys&amp;profile=%d",
		"/admin?section=import&amp;profile=%d", "/admin?section=relay&amp;profile=%d"} {
		require.Contains(t, body, `href="`+fmt.Sprintf(tab, p.ID)+`"`)
	}
	require.NotContains(t, body, `href="/account/keys"`) // context tabs never point at the own profile
	for _, section := range []string{"keys", "import", "relay", "account"} {
		w := taskRequest139(a, "GET", fmt.Sprintf("/admin?section=%s&profile=%d", section, p.ID), a.cookie, nil, nil)
		require.Equal(t, 200, w.Code, section)
		require.Contains(t, w.Body.String(), "you are operating as administrator", section)
	}
	account := taskRequest139(a, "GET", fmt.Sprintf("/admin?section=account&profile=%d", p.ID), a.cookie, nil, nil).Body.String()
	for _, want := range []string{"account-token", "account-role", "delete-review", "owner160"} {
		require.Contains(t, account, want)
	}
	require.NotContains(t, account, `name="confirm_password"`) // the root's own password card is not shown for others

	// Owner and ordinary users never reach the context of another account.
	for name, cookie := range map[string]string{"owner": owner, "ordinary": ordinary, "visitor": ""} {
		for _, path := range []string{"/admin?profile=1", "/admin?section=account&profile=1", "/admin?section=keys&profile=1"} {
			w := taskRequest139(a, "GET", path, cookie, nil, nil)
			if cookie != "" {
				require.Equal(t, 403, w.Code, name+" "+path)
			}
			// The visitor receives the sign-in document; nobody receives the context of another account.
			for _, hidden := range []string{"account-token", "account-role", "you are operating", "delete-review", `name="display_name"`} {
				require.NotContains(t, w.Body.String(), hidden, name+" "+path)
			}
		}
	}
	require.Equal(t, 403, taskRequest139(a, "GET", fmt.Sprintf("/admin?profile=%d", otherID), owner, nil, nil).Code)
	require.Equal(t, 403, taskRequest139(a, "GET", base, ordinary, nil, nil).Code)
}

// TestAccountsList126 lists accounts only for the administrator.
func TestAccountsList126(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	for _, path := range []string{"/admin", "/admin?section=profiles"} {
		w := taskRequest139(a, "GET", path, a.cookie, nil, nil)
		require.Equal(t, 200, w.Code, path)
		body := w.Body.String()
		require.Contains(t, body, `class="account-rows"`)
		require.Contains(t, body, `action="/admin/profile/create"`)
		require.Contains(t, body, fmt.Sprintf(`href="/admin?profile=%d"`, p.ID))
		require.Contains(t, body, "Open my account")
		require.Contains(t, body, "Administration") // menu group
		denied := taskRequest139(a, "GET", path, owner, nil, nil)
		require.Equal(t, 403, denied.Code, path)
		for _, hidden := range []string{"account-rows", "/admin/profile/create", "owner160"} {
			require.NotContains(t, denied.Body.String(), hidden, path)
		}
	}
	// The menu offers the group only to the administrator.
	require.NotContains(t, taskRequest139(a, "GET", "/account", owner, nil, nil).Body.String(), "nav-admin-title")
	require.Contains(t, taskRequest139(a, "GET", "/account", a.cookie, nil, nil).Body.String(), "nav-admin-title")
}

// TestProfileCardsAndFragments126 keeps the POST contracts: redirects land on the right tab, a
// script gets only the card with a fresh one-use ticket, and a refusal never becomes a fragment.
func TestProfileCardsAndFragments126(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	partial := map[string]string{"X-TuneTrail-Partial": "1"}
	tokens := func(cookie, path string) url.Values {
		page := taskRequest139(a, "GET", path, cookie, nil, nil)
		require.Equal(t, 200, page.Code, path)
		return url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")},
			"profile_slug": {p.Slug}}
	}
	route := fmt.Sprintf("/admin/profile/%d/", p.ID)

	// Without JavaScript the owner returns to the tab where the action was made.
	form := tokens(owner, "/account/keys")
	form.Set("label", "Plain key")
	out := taskRequest139(a, "POST", route+"keys/create", owner, form, nil)
	require.Equal(t, 303, out.Code)
	require.Equal(t, "/account/keys", out.Header().Get("Location"))
	require.Contains(t, taskRequest139(a, "GET", "/account/keys", owner, nil, nil).Body.String(), "Plain key")
	// A root operating the same account returns to the administration context.
	rootForm := tokens(a.cookie, fmt.Sprintf("/admin?section=keys&profile=%d", p.ID))
	rootForm.Set("label", "Root key")
	out = taskRequest139(a, "POST", route+"keys/create", a.cookie, rootForm, nil)
	require.Equal(t, 303, out.Code)
	require.Equal(t, fmt.Sprintf("/admin?section=keys&profile=%d", p.ID), out.Header().Get("Location"))

	// With JavaScript only the card comes back, with a fresh single-use ticket.
	form = tokens(owner, "/account/keys")
	form.Set("label", "Script key")
	out = taskRequest139(a, "POST", route+"keys/create", owner, form, partial)
	require.Equal(t, 200, out.Code, out.Body.String())
	require.Equal(t, "profile-keys", out.Header().Get("X-TuneTrail-Fragment"))
	require.Equal(t, "no-store", out.Header().Get("Cache-Control"))
	require.NotEmpty(t, out.Header().Get("X-TuneTrail-Submission"))
	require.NotContains(t, out.Body.String(), "<html")
	require.Contains(t, out.Body.String(), `id="profile-keys"`)
	require.Contains(t, out.Body.String(), "Script key")
	require.Contains(t, out.Body.String(), "Changes saved.")
	// The spent ticket is refused and the refusal is a whole document, never a fragment.
	replay := taskRequest139(a, "POST", route+"keys/create", owner, form, partial)
	require.GreaterOrEqual(t, replay.Code, 400)
	require.Empty(t, replay.Header().Get("X-TuneTrail-Fragment"))

	// A validation error stays inside the affected card.
	form = tokens(owner, "/account/profile")
	form.Set("display_name", "")
	form.Set("slug", p.Slug)
	form.Set("require_login_present", "1")
	out = taskRequest139(a, "POST", route+"save", owner, form, partial)
	require.Equal(t, 400, out.Code)
	require.Equal(t, "profile-identity", out.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, out.Body.String(), `role="alert"`)
	require.False(t, strings.Contains(out.Body.String(), "<html"))

	// The owner cannot act on another profile; the refusal is not a card.
	forged := tokens(owner, "/account/profile")
	forged.Set("profile_slug", "default")
	forged.Set("display_name", "Hijack")
	forged.Set("slug", "default")
	forged.Set("require_login_present", "1")
	out = taskRequest139(a, "POST", "/admin/profile/1/save", owner, forged, partial)
	require.Equal(t, 403, out.Code)
	require.Empty(t, out.Header().Get("X-TuneTrail-Fragment"))
	var name string
	require.NoError(t, a.store.QueryRow(`SELECT display_name FROM listening_profiles WHERE id=1`).Scan(&name))
	require.NotEqual(t, "Hijack", name)
	// Root tools are refused for the owner even with valid tokens (account-role on its own profile).
	roleForm := tokens(owner, "/account/profile")
	roleForm.Set("role", "admin")
	require.Equal(t, 403, taskRequest139(a, "POST", route+"account-role", owner, roleForm, partial).Code)
}
