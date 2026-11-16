package webassets

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocalFontGeneration guards font dependency hashes and private license boundaries.
func TestLocalFontGeneration(t *testing.T) {
	dir := distribution(t)
	first, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	font, ok := first.Asset("fonts/Jost-5.woff2")
	if !ok || font.ContentType != "font/woff2" {
		t.Fatal("missing WOFF2 MIME", font)
	}
	css, _ := first.Asset("ui.css")
	body, _ := io.ReadAll(css.Reader())
	if !strings.Contains(string(body), "fonts/Jost-5.woff2?v="+font.Digest) || strings.Contains(string(body), "fonts.googleapis.com") {
		t.Fatal("font dependency not local and versioned")
	}
	for _, family := range []string{"Jost", "LeagueSpartan", "Inter"} {
		if _, ok := first.Asset("fonts/" + family + "-OFL.txt"); ok {
			t.Fatal("license exposed", family)
		}
	}
	name := filepath.Join(dir, "fonts/Jost-5.woff2")
	if err := os.WriteFile(name, []byte("changed font representation"), 0644); err != nil {
		t.Fatal(err)
	}
	next, err := Open(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	changed, _ := next.Asset("ui.css")
	if changed.Digest == css.Digest {
		t.Fatal("font change must invalidate CSS")
	}
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(t.Context(), dir); err == nil {
		t.Fatal("missing font must reject generation")
	}
}
