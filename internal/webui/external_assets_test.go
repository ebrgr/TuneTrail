package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestExternalAssetExposure verifies file-only routes and stable snapshot headers before ISSUE-106.
func TestExternalAssetExposure(t *testing.T) {
	h := Assets()
	for _, path := range []string{"/", "/icons/", "/icons/LICENSE", "/templates/live.html", "/brand/icon.png", "/../templates/live.html"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code == 200 {
			t.Fatalf("exposed %s", path)
		}
	}
	for _, path := range []string{"/ui.css", "/admin.js", "/icons/House.svg", "/themes.css", "/themes.json"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("no nosniff for %s", path)
		}
		if w.Header().Get("Cache-Control") != "" || w.Header().Get("Last-Modified") != "" {
			t.Fatalf("changed cache contract for %s", path)
		}
		if path == "/admin.js" && !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
			t.Fatal("script MIME")
		}
	}
}
