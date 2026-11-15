package webassets

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// rewriteManifest replaces one manifest entry in a disposable distribution.
func rewriteManifest(t *testing.T, dir, key string, value any) {
	t.Helper()
	p := filepath.Join(dir, "manifest.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err = json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest[key] = value
	if b, err = json.Marshal(manifest); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(p, b, 0644); err != nil {
		t.Fatal(err)
	}
}

// TestStylesheetBundleJoinsPartsInOrder checks ordering and the newline added after a part without one (ISSUE-068).
func TestStylesheetBundleJoinsPartsInOrder(t *testing.T) {
	dir := distribution(t)
	if err := os.MkdirAll(filepath.Join(dir, "css/test/nested"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"css/test/b.css": ".b{}", "css/test/nested/a.css": ".a{}\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	rewriteManifest(t, dir, "public/static/ui.css", []string{"css/test/b.css", "css/test/nested/a.css"})
	s, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.read("public/static/ui.css")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != ".b{}\n.a{}\n" {
		t.Fatalf("bundle = %q", got)
	}
}

// TestShippedBundleDeclaresLayerOrderFirst keeps the @layer order statement ahead of every layer block.
func TestShippedBundleDeclaresLayerOrderFirst(t *testing.T) {
	s, err := Open(t.Context(), distribution(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"public/static/ui.css", "public/static/admin.css"} {
		got, err := s.read(name)
		if err != nil {
			t.Fatal(err)
		}
		order := bytes.Index(got, []byte("@layer reset, tokens, base, layout, components, pages, utilities;"))
		block := bytes.Index(got, []byte(" {\n"))
		first := bytes.Index(got, []byte("@layer "))
		if order < 0 || order != first || order > block {
			t.Fatalf("%s: layer order statement is not the first layer rule", name)
		}
	}
}

// TestNestedPartURLResolvesFromServedStylesheet documents that url() is relative to ui.css, not to the part's folder.
func TestNestedPartURLResolvesFromServedStylesheet(t *testing.T) {
	for _, tc := range []struct {
		url string
		ok  bool
	}{{"icons/Music2.svg", true}, {"/web-assets/icons/Music2.svg", true}, {"../../icons/Music2.svg", false}} {
		dir := distribution(t)
		part := filepath.Join(dir, "css/ui/pages/home.css")
		data, err := os.ReadFile(part)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(part, append(data, []byte(".x{mask:url('"+tc.url+"')}\n")...), 0644); err != nil {
			t.Fatal(err)
		}
		s, err := Open(t.Context(), dir)
		if !tc.ok {
			if err == nil {
				t.Fatalf("%s: path outside the public root accepted", tc.url)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		a, _ := s.Asset("ui.css")
		icon, _ := s.Asset("icons/Music2.svg")
		if !bytes.Contains(a.data, []byte(`url("icons/Music2.svg?v=`+icon.Digest+`")`)) {
			t.Fatalf("%s: not resolved from the served stylesheet", tc.url)
		}
	}
}

// TestStylesheetPartChangesVersionedURL proves editing any one part yields a new immutable URL.
func TestStylesheetPartChangesVersionedURL(t *testing.T) {
	dir := distribution(t)
	before, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	part := filepath.Join(dir, "css/ui/utilities.css")
	data, err := os.ReadFile(part)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(part, append(data, []byte(".changed{color:red}\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	old, _ := before.AssetURL("/web-assets", "ui.css")
	next, _ := after.AssetURL("/web-assets", "ui.css")
	if old == next {
		t.Fatalf("URL kept after part change: %s", next)
	}
	adminBefore, _ := before.AssetURL("/web-assets", "admin.css")
	adminAfter, _ := after.AssetURL("/web-assets", "admin.css")
	if adminBefore != adminAfter {
		t.Fatal("unrelated stylesheet changed URL")
	}
}

// TestManifestSourcesRejectsInvalidBundles keeps lists limited to non-empty, duplicate-free public CSS.
func TestManifestSourcesRejectsInvalidBundles(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"public/static/ui.js", `["js/a.js","js/b.js"]`},
		{"public/templates/page.html", `["templates/public/page.html"]`},
		{"public/static/ui.css", `[]`},
		{"public/static/ui.css", `["css/ui/base.css","css/ui/base.css"]`},
		{"public/static/ui.css", `{"a":"b"}`},
		{"public/static/ui.css", `3`},
	} {
		if _, err := manifestSources(tc.key, json.RawMessage(tc.value)); err == nil {
			t.Fatalf("accepted %s: %s", tc.key, tc.value)
		}
	}
	if !validEntry("public/static/ui.css", "css/ui/pages/home.css") {
		t.Fatal("nested CSS part refused")
	}
	if validEntry("public/static/ui.css", "js/ui.js") {
		t.Fatal("non-CSS part accepted in a stylesheet")
	}
}
