package i18n

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// TestResolution checks browser preferences, invalid inputs and explicit choices.
func TestResolution(t *testing.T) {
	c, err := Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ header, cookie, tag string }{
		{"pt-BR,pt;q=0.9", "", "pt-BR"}, {"pt", "", "pt-BR"}, {"de-DE", "", "en"}, {"garbage_!", "", "en"}, {"pt-BR", "en", "en"}, {"en", "pt-BR", "pt-BR"}, {"en", "invalid", "en"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", tc.header)
		if tc.cookie != "" {
			r.AddCookie(&http.Cookie{Name: cookieName, Value: tc.cookie})
		}
		if got := c.Resolve(r).Tag; got != tc.tag {
			t.Errorf("%q %q: %s", tc.header, tc.cookie, got)
		}
	}
}

// TestCatalogValidation checks absent files, placeholders, fallback and external edits.
func TestCatalogValidation(t *testing.T) {
	dir := t.TempDir()
	en, err := os.ReadFile("../../langs/en.json")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := os.ReadFile("../../langs/pt-BR.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "en.json"), en, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Load(dir); err == nil {
		t.Fatal("missing catalog accepted")
	}
	var m map[string]any
	if err = json.Unmarshal(pt, &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "profiles.title")
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "pt-BR.json"), data, 0600)
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "pt-BR")
	if got := c.Resolve(r).T("profiles.title"); got != "Choose a listening profile" {
		t.Fatalf("missing translation fallback: %q", got)
	}
	m["profiles.intro"] = "Olá {{.Name}}"
	data, _ = json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "pt-BR.json"), data, 0600)
	if _, err = Load(dir); err == nil {
		t.Fatal("mismatched placeholder accepted")
	}
	os.WriteFile(filepath.Join(dir, "pt-BR.json"), []byte("{"), 0600)
	if _, err = Load(dir); err == nil {
		t.Fatal("invalid catalog accepted")
	}
}

// TestPreferenceForm validates the anonymous preference route without following redirects.
func TestPreferenceForm(t *testing.T) {
	c, err := Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		form   string
		status int
	}{
		{"language=pt-BR&return_to=%2Fadmin", 303}, {"language=auto&return_to=%2F", 303}, {"language=xx&return_to=%2F", 400}, {"language=pt-BR&return_to=https%3A%2F%2Fevil.test", 400}, {"language=pt-BR&language=en&return_to=%2F", 400},
	} {
		r := httptest.NewRequest("POST", "http://example.test/language", strings.NewReader(tc.form))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		c.SetLanguage(w, r)
		if w.Code != tc.status {
			t.Errorf("%s: %d", tc.form, w.Code)
		}
		if tc.status == 303 && len(w.Result().Cookies()) != 1 {
			t.Error("missing cookie")
		}
	}
}

// TestCatalogRejectsFIFO ensures special files never block startup.
func TestCatalogRejectsFIFO(t *testing.T) {
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "en.json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("FIFO catalog accepted")
	}
}

// TestMessageVariablesAreEscapedByHTMLTemplate checks parameter and translation boundaries.
func TestMessageVariablesAreEscapedByHTMLTemplate(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"en.json", "pt-BR.json"} {
		data, err := os.ReadFile(filepath.Join("../../langs", name))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err = json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		m["greeting"] = "Hello {{.Name}}"
		if name == "pt-BR.json" {
			m["greeting"] = "Olá {{.Name}}"
		}
		data, err = json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "pt-BR")
	translated := c.Resolve(r).Format("greeting", map[string]any{"Name": "<script>alert(1)</script>"}, nil)
	tpl := template.Must(template.New("page").Parse(`<p>{{.}}</p>`))
	var out bytes.Buffer
	if err = tpl.Execute(&out, translated); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "<script>") || !strings.Contains(out.String(), "&lt;script&gt;") {
		t.Fatal("translation did not escape HTML")
	}
}
