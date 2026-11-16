package engine

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestArtworkDropContract142 checks the authenticated public markup and the existing image task contract.
func TestArtworkDropContract142(t *testing.T) {
	a := taskApp139(t)
	adminPage := taskRequest139(a, "GET", "/second/album/11", a.cookie, nil, nil)
	require.Equal(t, 200, adminPage.Code, adminPage.Body.String())
	require.Contains(t, adminPage.Body.String(), "data-artwork-drop")
	require.Contains(t, adminPage.Body.String(), "data-artwork-task")
	// The drop handler resolves the task link page-wide: exactly one must exist for an album page.
	require.Equal(t, 1, strings.Count(adminPage.Body.String(), "data-artwork-task"))

	visitorPage := taskRequest139(a, "GET", "/second/album/11", "", nil, nil)
	require.Equal(t, 200, visitorPage.Code, visitorPage.Body.String())
	require.NotContains(t, visitorPage.Body.String(), "data-artwork-drop")
	require.NotContains(t, visitorPage.Body.String(), "data-artwork-task")

	task := taskRequest139(a, "GET", "/second/manage/album/11/image?partial=1", a.cookie, nil,
		map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"})
	require.Equal(t, 200, task.Code, task.Body.String())
	require.Contains(t, task.Body.String(), `enctype="multipart/form-data"`)
	require.Contains(t, task.Body.String(), `data-task-cancel`)
	require.Contains(t, task.Body.String(), `name="image"`)
}

// TestArtworkPreviewLifecycle142 checks the client-side safety hooks without requiring a browser runner.
func TestArtworkPreviewLifecycle142(t *testing.T) {
	source, err := os.ReadFile("../assets/js/media-dialog.js")
	require.NoError(t, err)
	code := string(source)
	for _, marker := range []string{
		"pendingArtworkFile",
		"URL.revokeObjectURL",
		"Preview ready. Confirm to replace the artwork.",
		"data-artwork-drop",
		"data-task-cancel",
	} {
		require.True(t, strings.Contains(code, marker), "missing artwork lifecycle marker %q", marker)
	}
	// Regression: the link sits in the actions nav, not inside the cover's .media-heading header.
	require.NotContains(t, code, ".media-heading')?.querySelector('a[data-artwork-task]')")
}
