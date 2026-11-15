package engine

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdminSidebarAvatarVariants renders the shared sidebar in administration for a
// profile icon and a profile image (ISSUE-122): bundled SVG and the shared fallback hooks.
func TestAdminSidebarAvatarVariants(t *testing.T) {
	a := newProfileTestApp(t)
	sidebar := func() string {
		w := a.request(http.MethodGet, "/admin?profile=2", "", "", true)
		require.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		start := strings.Index(body, `<nav class="sidebar"`)
		require.GreaterOrEqual(t, start, 0)
		return body[start : start+strings.Index(body[start:], "</nav>")]
	}

	require.NoError(t, a.store.Exec(`UPDATE listening_profiles SET avatar_icon='radio' WHERE id=2`))
	icon := sidebar()
	require.Contains(t, icon, `class="lucide lucide-radio"`)
	require.NotContains(t, icon, "admin-avatar-icon")

	require.NoError(t, a.store.Exec(`UPDATE listening_profiles SET avatar_icon=NULL, avatar_image_id='abc123' WHERE id=2`))
	image := sidebar()
	require.Contains(t, image, `src="/image/abc123/128x128.webp"`)
	require.Contains(t, image, "data-avatar-image")
	require.Contains(t, image, "data-avatar-fallback")
}
