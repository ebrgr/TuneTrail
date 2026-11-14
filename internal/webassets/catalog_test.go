package webassets

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCatalogTracksDependencies verifies content identity and dependency-specific invalidation.
func TestCatalogTracksDependencies(t *testing.T) {
	dir := distribution(t)
	first, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ui.css", "admin.css", "themes.css", "theme.js", "themes.json"} {
		a, ok := first.Asset(name)
		if !ok {
			t.Fatal(name)
		}
		body, err := io.ReadAll(a.Reader())
		if err != nil {
			t.Fatal(err)
		}
		b, _ := again.Asset(name)
		if a.Digest != digest(body) || a.Digest != b.Digest {
			t.Fatal("unstable representation", name)
		}
	}
	icon := filepath.Join(dir, "icons/Music2.svg")
	if err := os.WriteFile(icon, []byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle r="3"/></svg>`), 0644); err != nil {
		t.Fatal(err)
	}
	next, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"icons/Music2.svg", "admin.css", "ui.css", "theme.js"} {
		a, _ := first.Asset(name)
		b, _ := next.Asset(name)
		changed := name == "icons/Music2.svg" || name == "admin.css"
		if (a.Digest != b.Digest) != changed {
			t.Fatal("incorrect invalidation", name)
		}
	}
	cssAsset, _ := next.Asset("admin.css")
	body, err := io.ReadAll(cssAsset.Reader())
	if err != nil {
		t.Fatal(err)
	}
	changedIcon, _ := next.Asset("icons/Music2.svg")
	if !strings.Contains(string(body), `url("icons/Music2.svg?v=`+changedIcon.Digest+`")`) {
		t.Fatal("unversioned dependency")
	}
	if _, ok := next.Asset("icons/LICENSE"); ok {
		t.Fatal("license exposed")
	}
}

// TestCSSDependencyPolicy checks CSS tokens, escaped references and unsupported imports.
func TestCSSDependencyPolicy(t *testing.T) {
	s, err := Open(t.Context(), distribution(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		source string
		bad    bool
	}{
		{`a{mask:url('/web-assets/icons/Music2.svg')}`, false},
		{`a{mask:uRl(icons/Music2.svg#shape)}`, false},
		{`a{mask:url('icons/\4d usic2.svg')}`, false},
		{`/* @import 'missing.css'; url(missing) */ a{content:"url(missing)"}`, false},
		{`a{fill:url(#gradient);background:url(data:image/svg+xml;base64,AAAA)}`, false},
		{`a{background:url(https://example.com/image.png)}`, false},
		{`a{background:image-set("icons/Music2.svg" 1x)}`, true},
		{`a{background:-webkit-image-set("icons/Music2.svg" 1x)}`, true},
		{`a{src:src("font.woff2")}`, true},
		{`a{content:"image-set('unchanged')"}`, false},
		{`@import 'ui.css';`, true},
		{`@\69mport 'ui.css';`, true},
		{`a{background:url(ui.css)}`, true},
		{`a{mask:url(icons/missing.svg)}`, true},
		{`a{mask:url(icons/Music2.svg?old=1)}`, true},
		{`a{mask:url(../../private.svg)}`, true},
	} {
		t.Run(tc.source, func(t *testing.T) {
			_, err := s.rewriteCSS("admin.css", []byte(tc.source))
			if (err != nil) != tc.bad {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestAssetTemplateValidation catches invalid literals even in branches that never execute.
func TestAssetTemplateValidation(t *testing.T) {
	asset, err := AssetFunc("/web-assets")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body string
		bad  bool
	}{
		{`{{asset "ui.css"}}`, false},
		{`{{if false}}{{asset "missing.css"}}{{end}}`, true},
		{`{{if false}}{{else}}{{asset "icons/LICENSE"}}{{end}}`, true},
		{`{{asset .Name}}`, true},
		{`{{if false}}{{(asset "missing.css").Field}}{{end}}`, true},
		{`{{if (asset "missing.css")}}bad{{end}}`, true},
		{`{{"ui.css" | asset}}`, true},
	} {
		parsed, err := template.New("test").Funcs(template.FuncMap{"asset": asset}).Parse(tc.body)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateTemplates(parsed); (err != nil) != tc.bad {
			t.Fatalf("%s: %v", tc.body, err)
		}
	}
}

// TestDerivedCSSBudget enforces the limit after URL expansion, not only on source bytes.
func TestDerivedCSSBudget(t *testing.T) {
	dir := distribution(t)
	rule := `a{mask:url(icons/Music2.svg)}`
	source := strings.Repeat(rule, maxFileBytes/len(rule))
	if err := os.WriteFile(filepath.Join(dir, "css/admin.css"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), dir); err == nil {
		t.Fatal("expanded CSS exceeded the generation budget")
	}
}
