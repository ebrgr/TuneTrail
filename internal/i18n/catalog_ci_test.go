package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestCatalogCompleteness keeps shipped translations and literal template keys synchronized in CI.
func TestCatalogCompleteness(t *testing.T) {
	catalogs := make([]map[string]json.RawMessage, 2)
	for i, name := range []string{"en.json", "pt-BR.json"} {
		data, err := os.ReadFile(filepath.Join("../../langs", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &catalogs[i]); err != nil {
			t.Fatal(err)
		}
	}
	for key := range catalogs[0] {
		if _, ok := catalogs[1][key]; !ok {
			t.Errorf("Portuguese catalog missing %s", key)
		}
	}
	for key := range catalogs[1] {
		if _, ok := catalogs[0][key]; !ok {
			t.Errorf("English catalog missing %s", key)
		}
	}
	expr := regexp.MustCompile(`\b(?:tr|tr1|tr2|trn|trnp|quantity|unit)\s+"([^"]+)"`)
	for _, dir := range []string{"../../internal/webui/templates", "../../internal/adminui/templates"} {
		paths, err := filepath.Glob(filepath.Join(dir, "*.html"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range expr.FindAllSubmatch(data, -1) {
				key := string(match[1])
				if _, ok := catalogs[0][key]; !ok {
					t.Errorf("%s references missing key %s", path, key)
				}
			}
		}
	}
	if _, err := Load("../../langs"); err != nil {
		t.Fatal(err)
	}
}
