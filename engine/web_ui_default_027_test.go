package engine

import (
	"net/http"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/stretchr/testify/require"
)

// TestGoModeDoesNotExposeReact ensures the Go dispatcher has no SPA fallback.
func TestGoModeDoesNotExposeReact(t *testing.T) {
	require.NoError(t, cfg.Load(func(key string) string {
		if key == cfg.WEB_UI_ENV {
			return "go"
		}
		return ""
	}, "test"))
	app := newProfileTestApp(t)
	for _, path := range []string{"/theme-helper", "/index.html", "/assets/unknown.js", "/default/unknown"} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			response := app.request(method, path, "", "", false)
			require.Equal(t, http.StatusNotFound, response.Code, "%s %s", method, path)
		}
	}
	for _, path := range []string{"/", "/default", "/favicon.svg", "/web-assets/ui.css", "/apis/web/v1/profiles"} {
		response := app.request(http.MethodGet, path, "", "", false)
		require.Equal(t, http.StatusOK, response.Code, "GET %s", path)
	}
}
