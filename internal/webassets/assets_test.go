package webassets

import (
	"context"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// distribution copies only manifest-owned fixture files; tests never mutate checkout resources.
func distribution(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../assets")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	b, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]json.RawMessage
	if err = json.Unmarshal(b, &manifest); err != nil {
		t.Fatal(err)
	}
	names := []string{"manifest.json"}
	for key, value := range manifest {
		parts, err := manifestSources(key, value)
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, parts...)
	}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(dir, name)
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(dest, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestSnapshotKeepsGeneration proves that serving cannot mix files modified after loading.
func TestSnapshotKeepsGeneration(t *testing.T) {
	dir := distribution(t)
	s, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.read("public/static/ui.css")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "css/ui/base.css"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	after, err := s.read("public/static/ui.css")
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("generation changed")
	}
	before[0] ^= 1
	again, _ := s.read("public/static/ui.css")
	if string(before) == string(again) {
		t.Fatal("caller mutated snapshot")
	}
	next, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if next.Hash == s.Hash {
		t.Fatal("new generation kept old hash")
	}
}

// TestSnapshotRejectsBrokenDistribution exercises early failures and confinement, including escaping links.
func TestSnapshotRejectsBrokenDistribution(t *testing.T) {
	for _, mode := range []string{"missing", "directory", "oversize", "escape", "manifest"} {
		t.Run(mode, func(t *testing.T) {
			dir := distribution(t)
			p := filepath.Join(dir, "css/ui/base.css")
			switch mode {
			case "missing":
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(p, 0755); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(p, []byte(strings.Repeat("x", maxFileBytes+1)), 0644); err != nil {
					t.Fatal(err)
				}
			case "escape":
				outside := filepath.Join(t.TempDir(), "external.css")
				if err := os.WriteFile(outside, []byte("outside"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, p); err != nil {
					t.Fatal(err)
				}
			case "manifest":
				if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{}`), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := Open(t.Context(), dir); err == nil {
				t.Fatal("invalid distribution accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Open(ctx, distribution(t)); err == nil {
		t.Fatal("canceled load succeeded")
	}
}

// TestExplicitRootNeverFallsBack checks the documented root override contract.
func TestExplicitRootNeverFallsBack(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "missing")
	t.Setenv("TUNETRAIL_WEB_DIR", absent)
	p, err := Locate()
	if err != nil {
		t.Fatal(err)
	}
	if p != absent {
		t.Fatalf("fallback: %s", p)
	}
	if _, err := Open(t.Context(), p); err == nil {
		t.Fatal("missing override accepted")
	}
}

// TestManifestCategories refuses path traversal and public/private source aliases.
func TestManifestCategories(t *testing.T) {
	for _, entry := range [][2]string{
		{"public/templates/evil.html", "templates/admin/admin.html"},
		{"public/static/ui.css", "templates/admin/admin.html"},
		{"public/static/ui.css", "manifest.json"},
		{"public/static/tool.sh", "js/tool.sh"},
		{"public/palettes.json", "themes/other.json"},
		{"public/static/../ui.css", "css/ui.css"},
		{"public/static/ui.css", "/etc/secret"},
	} {
		if validEntry(entry[0], entry[1]) {
			t.Fatalf("invalid manifest accepted: %v", entry)
		}
	}
}

// TestInternalSymlinkCannotExposePrivateTemplate checks category isolation inside the root.
func TestInternalSymlinkCannotExposePrivateTemplate(t *testing.T) {
	dir := distribution(t)
	p := filepath.Join(dir, "css/ui/base.css")
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../templates/admin/admin.html", p); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), dir); err == nil {
		t.Fatal("private template symlink accepted")
	}
}

// TestTemplateReferences checks missing partials in both branches and reports their source.
func TestTemplateReferences(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"direct", `{{template "absent" .}}`},
		{"if", `{{if .}}{{template "absent" .}}{{end}}`},
		{"range", `{{range .}}{{template "absent" .}}{{end}}`},
		{"with", `{{with .}}{{template "absent" .}}{{end}}`},
		{"if else", `{{if .}}{{else}}{{template "absent" .}}{{end}}`},
		{"range else", `{{range .}}{{else}}{{template "absent" .}}{{end}}`},
		{"with else", `{{with .}}{{else}}{{template "absent" .}}{{end}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := template.New("caller").Parse(tc.body)
			if err != nil {
				t.Fatal(err)
			}
			err = ValidateTemplates(parsed)
			if err == nil || err.Error() != `template "caller": undefined or bodyless template "absent"` {
				t.Fatalf("unexpected diagnostic: %v", err)
			}
		})
	}
	parsed, err := template.New("root").Parse(`{{define "present"}}ok{{end}}{{template "present" .}}`)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateTemplates(parsed); err != nil {
		t.Fatal(err)
	}
}

// TestTemplateReferenceWithoutBody rejects a call to an associated but unparsed root.
func TestTemplateReferenceWithoutBody(t *testing.T) {
	parsed := template.New("admin")
	if _, err := parsed.New("caller").Parse(`{{template "admin" .}}`); err != nil {
		t.Fatal(err)
	}
	err := ValidateTemplates(parsed)
	if err == nil || err.Error() != `template "caller": undefined or bodyless template "admin"` {
		t.Fatalf("unexpected diagnostic: %v", err)
	}
}
