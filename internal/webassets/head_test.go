package webassets

import "testing"

// TestHeadFuncKinds guards the shared head: only known page kinds render.
func TestHeadFuncKinds(t *testing.T) {
	for _, kind := range []string{"live", "preview", "admin"} {
		h, err := HeadFunc(kind, "Title")
		if err != nil || h.Kind != kind || h.Title != "Title" {
			t.Fatalf("HeadFunc(%q) = %+v, %v", kind, h, err)
		}
	}
	if _, err := HeadFunc("other", "Title"); err == nil {
		t.Fatal("unknown kind must fail rendering")
	}
}

// TestSharedTemplatesMountInBothSets allows templates/shared only as a template entry.
func TestSharedTemplatesMountInBothSets(t *testing.T) {
	for key, want := range map[string]bool{
		"public/templates/head.html": true,
		"admin/templates/head.html":  true,
		"public/static/head.html":    false,
		"public/templates/head.css":  false,
	} {
		name := "templates/shared/head.html"
		if key == "public/templates/head.css" {
			name = "templates/shared/head.css"
		}
		if got := validEntry(key, name); got != want {
			t.Errorf("validEntry(%q, %q) = %v, want %v", key, name, got, want)
		}
	}
}
