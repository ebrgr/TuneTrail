package engine

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAvatarColorForm202 covers the avatar background dropdown end to end: every palette entry is stored and shown,
// anything else is refused without changing the stored colour, a form without the field keeps the colour, an empty
// value returns to the theme colour and "Use initials" keeps it (ISSUE-202).
func TestAvatarColorForm202(t *testing.T) {
	a := taskApp139(t)
	page := func(path string) string {
		w := taskRequest139(a, "GET", path, a.cookie, nil, nil)
		require.Equal(t, 200, w.Code)
		return w.Body.String()
	}
	post := func(action string, fields url.Values) int {
		w := taskRequest139(a, "GET", "/admin?profile=2", a.cookie, nil, nil)
		fields.Set("csrf_token", taskField139(t, w, "csrf_token"))
		fields.Set("submission_token", taskField139(t, w, "submission_token"))
		fields.Set("profile_slug", "second")
		return taskRequest139(a, "POST", "/admin/profile/2/"+action, a.cookie, fields, nil).Code
	}
	stored := func() string {
		p, err := a.store.GetListeningProfileByID(context.Background(), 2)
		require.NoError(t, err)
		if p.AvatarColor == nil {
			return "<null>"
		}
		return *p.AvatarColor
	}

	// The dropdown lists the default and every palette entry, and starts on the default.
	body := page("/admin?profile=2")
	require.Contains(t, body, `<select name="color">`)
	require.Contains(t, body, `<option value="" selected>`)
	for _, hex := range []string{"#000000", "#374151", "#b91c1c", "#c2410c", "#a16207", "#15803d", "#0f766e", "#1d4ed8", "#6d28d9", "#be185d", "#d1d5db", "#ffffff"} {
		require.Contains(t, body, `<option value="`+hex+`"`)
	}
	require.NotContains(t, body, "data-avatar-color")

	// Every accepted value is stored, preselected, and drawn on the avatar.
	for _, hex := range []string{"#000000", "#1d4ed8", "#ffffff"} {
		require.Equal(t, 303, post("avatar", url.Values{"icon": {"disc"}, "color": {hex}}), hex)
		require.Equal(t, hex, stored())
		body = page("/admin?profile=2")
		require.Contains(t, body, `<option value="`+hex+`" selected>`, hex)
		require.Contains(t, body, `data-avatar-color="`+hex+`"`, hex)
	}

	// Values outside the palette are refused and leave the stored colour untouched.
	for _, bad := range []string{"red", "#123456", "#FFFFFF", "#00000", "url(x)", `#000000" onload="x`} {
		require.Equal(t, 400, post("avatar", url.Values{"icon": {"disc"}, "color": {bad}}), bad)
		require.Equal(t, "#ffffff", stored(), bad)
		require.NotContains(t, page("/admin?profile=2"), "onload=", bad)
	}

	// A form that sends no colour keeps it; the palette never reaches the page unchecked.
	require.Equal(t, 303, post("avatar", url.Values{"icon": {"user"}}))
	require.Equal(t, "#ffffff", stored())

	// "Use initials" clears the icon and keeps the background.
	require.Equal(t, 303, post("avatar-clear", url.Values{}))
	require.Equal(t, "#ffffff", stored())
	require.NotContains(t, page("/admin?profile=2"), `<option value="user" selected>`)

	// An empty value returns to the theme colour.
	require.Equal(t, 303, post("avatar", url.Values{"icon": {""}, "color": {""}}))
	require.Equal(t, "<null>", stored())
	require.NotContains(t, page("/admin?profile=2"), "data-avatar-color")

	// A stored value outside the palette (bad data) is never written into a page.
	require.NoError(t, a.store.Exec(`UPDATE listening_profiles SET avatar_color='#abcdef' WHERE id=2`))
	body = page("/admin?profile=2")
	require.False(t, strings.Contains(body, `data-avatar-color="#abcdef"`))
}

// TestAvatarColorAPI202 gives the multipart avatar API the same rules: an absent field keeps the colour.
func TestAvatarColorAPI202(t *testing.T) {
	a := newProfileTestApp(t)
	put := func(fields map[string]string) (int, string) {
		w := a.upload(t, "PUT", "/apis/web/v1/profiles/second/avatar", fields, "", "", nil, true)
		return w.Code, w.Body.String()
	}
	code, body := put(map[string]string{"icon": "music", "color": "#be185d"})
	require.Equal(t, 200, code)
	require.Contains(t, body, `"avatar_color":"#be185d"`)

	code, body = put(map[string]string{"icon": "disc"})
	require.Equal(t, 200, code)
	require.Contains(t, body, `"avatar_color":"#be185d"`, "an absent colour field keeps the background")

	code, _ = put(map[string]string{"icon": "disc", "color": "#abcdef"})
	require.Equal(t, 400, code)

	code, body = put(map[string]string{"icon": "disc", "color": ""})
	require.Equal(t, 200, code)
	require.NotContains(t, body, "avatar_color")
}
