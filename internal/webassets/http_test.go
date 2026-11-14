package webassets

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAssetHTTPContracts exercises cache identity before conditionals, including error paths.
func TestAssetHTTPContracts(t *testing.T) {
	s, err := Current()
	if err != nil {
		t.Fatal(err)
	}
	a, ok := s.Asset("ui.css")
	if !ok {
		t.Fatal("missing fixture")
	}
	etag := `"` + a.Digest + `"`
	valid := "/ui.css?v=" + a.Digest
	handler := Handler("", true)
	for _, tc := range []struct {
		name, method, url, header, value string
		status                           int
		immutable                        bool
	}{
		{"plain", "GET", "/ui.css", "", "", 200, false},
		{"versioned", "GET", valid, "", "", 200, true},
		{"head", "HEAD", valid, "", "", 200, true},
		{"etag", "GET", valid, "If-None-Match", etag, 304, true},
		{"weak", "GET", valid, "If-None-Match", "W/" + etag, 304, true},
		{"list", "HEAD", valid, "If-None-Match", `"other", ` + etag, 304, true},
		{"star", "GET", "/ui.css", "If-None-Match", "*", 304, false},
		{"range", "GET", valid, "Range", "bytes=0-9", 206, true},
		{"bad range", "GET", valid, "Range", "bytes=99999999-", 416, false},
		{"precondition", "GET", valid, "If-Match", `"wrong"`, 412, false},
		{"old", "GET", "/ui.css?v=old", "If-None-Match", etag, 404, false},
		{"extra", "GET", valid + "&x=1", "", "", 404, false},
		{"duplicate", "GET", valid + "&v=" + a.Digest, "", "", 404, false},
		{"empty", "GET", "/ui.css?v=", "", "", 404, false},
		{"empty query", "GET", "/ui.css?", "", "", 404, false},
		{"missing", "GET", "/missing.css", "", "", 404, false},
		{"license", "GET", "/icons/LICENSE", "", "", 404, false},
		{"source", "GET", "/templates/live.html", "", "", 404, false},
		{"method", "POST", valid, "", "", 405, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.url, nil)
			if tc.header != "" {
				r.Header.Set(tc.header, tc.value)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			cache := w.Header().Get("Cache-Control")
			if strings.Contains(cache, "immutable") != tc.immutable {
				t.Fatal(cache)
			}
			if tc.status >= 400 {
				if cache != "no-store" || w.Header().Get("ETag") != "" {
					t.Fatal("cached error", w.Header())
				}
			} else if w.Header().Get("ETag") != etag {
				t.Fatal("etag", w.Header())
			}
			if (tc.method == "HEAD" || tc.status == 304) && w.Body.Len() != 0 {
				t.Fatal("unexpected body")
			}
			if w.Header().Get("Last-Modified") != "" {
				t.Fatal("invented timestamp")
			}
		})
	}
	// Both range validators are interpreted by ServeContent, not custom parsing.
	r := httptest.NewRequest("GET", valid, nil)
	r.Header.Set("Range", "bytes=0-9")
	r.Header.Set("If-Range", etag)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 206 || w.Body.Len() != 10 {
		t.Fatal("if-range", w.Code)
	}
	legacy := httptest.NewRecorder()
	Handler("", false).ServeHTTP(legacy, httptest.NewRequest("GET", valid, nil))
	if legacy.Code != 404 || legacy.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("versioned legacy alias")
	}
}
