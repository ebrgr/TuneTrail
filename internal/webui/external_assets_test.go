package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestExternalAssetExposure verifies file-only routes and stable snapshot headers with mandatory legacy revalidation.
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
		if w.Header().Get("Cache-Control") != "no-cache" || w.Header().Get("Last-Modified") != "" {
			t.Fatalf("changed cache contract for %s", path)
		}
		if path == "/admin.js" && !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
			t.Fatal("script MIME")
		}
	}
}

// TestDocumentAssetMetadata checks the URLs consumed by script and avoids duplicate context attributes.
func TestDocumentAssetMetadata(t *testing.T) {
	h, err := NewHandler(pageStore(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha?period=all_time&tz=UTC").Body.String()
	for _, name := range []string{"data-timezone=", "data-week-start=", "data-cover-fallback-url=", "data-themes-url="} {
		if strings.Count(body, name) != 1 {
			t.Fatalf("duplicate or missing %s", name)
		}
	}
	if !strings.Contains(body, `data-cover-fallback-url="/web-assets/icons/Disc3.svg?v=`) {
		t.Fatal("unversioned image fallback")
	}
}
