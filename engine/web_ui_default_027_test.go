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

// TestDocumentCachePrivacy checks the routed public/admin boundary, including error documents.
func TestDocumentCachePrivacy(t *testing.T) {
	app := newProfileTestApp(t)
	for _, tc := range []struct {
		path, cache string
		status      int
	}{
		{"/", "private, no-cache", 200},
		{"/default", "private, no-cache", 200},
		{"/default/search?q=example", "private, no-cache", 200},
		{"/missing", "no-store", 404},
		{"/default/unknown", "no-store", 404},
		{"/admin", "no-store", 200},
		{"/admin/settings", "no-store", 200},
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			response := app.request(method, tc.path, "", "", false)
			require.Equal(t, tc.status, response.Code, method+" "+tc.path)
			require.Equal(t, tc.cache, response.Header().Get("Cache-Control"), method+" "+tc.path)
			if tc.cache == "private, no-cache" {
				require.Contains(t, response.Header().Values("Vary"), "Cookie")
				require.Contains(t, response.Header().Values("Vary"), "Accept-Language")
				require.NotEmpty(t, response.Header().Get("X-TuneTrail-UI-Generation"))
			}
		}
	}
}
