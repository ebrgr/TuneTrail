package webui

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestPreviewRoutesAndEscaping(t *testing.T) {
	h, err := NewPreview()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/demo", "/luna", "/demo?state=empty", "/demo?state=error", "/demo?state=long", "/demo?state=broken"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "ZgotmplZ") {
			t.Fatalf("unsafe template value: %s", path)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/does-not-exist", nil))
	if w.Code != 404 {
		t.Fatal("unknown slug must not fall back")
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/demo?theme=%22%3E%3Cscript%3Ealert(1)%3C/script%3E&period=invalid", nil))
	if strings.Contains(w.Body.String(), "<script>alert(1)") {
		t.Fatal("query injection")
	}
	if !strings.Contains(w.Body.String(), `data-theme="yuu"`) {
		t.Fatal("invalid theme must fall back")
	}
	// html/template must escape profile names, independent of the HTTP fixtures.
	original := profiles[0].Name
	profiles[0].Name = `<script>alert("x")</script>`
	defer func() { profiles[0].Name = original }()
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/demo", nil))
	if strings.Contains(w.Body.String(), `<script>alert("x")`) {
		t.Fatal("profile HTML injection")
	}
	if !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("escaped text missing")
	}
}

// TestPalettesComplete verifies all bundled themes remain available without React.
func TestPalettesComplete(t *testing.T) {
	if err := ensureAssets(); err != nil {
		t.Fatal(err)
	}
	if len(palettes) != 12 {
		t.Fatal("expected all 12 palettes")
	}
	for _, name := range themeNames {
		if _, ok := palettes[name]; !ok {
			t.Fatalf("missing palette %s", name)
		}
	}
	for name, colors := range palettes {
		if len(colors) != 14 {
			t.Fatalf("%s: incomplete palette", name)
		}
		for _, key := range []string{"bg", "bgSecondary", "bgTertiary", "fg", "fgSecondary", "fgTertiary", "primary", "primaryDim", "accent", "accentDim", "error", "warning", "success", "info"} {
			if !regexp.MustCompile(`^#[0-9a-fA-F]{6}$`).MatchString(colors[key]) {
				t.Fatalf("%s: invalid %s", name, key)
			}
		}
	}
}
