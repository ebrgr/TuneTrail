package webui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
)

// TestTranslatedRewindAndPartialFailures checks populated content and safe failure messages.
func TestTranslatedRewindAndPartialFailures(t *testing.T) {
	s := pageStore(t)
	at := time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)
	if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: at}); err != nil {
		t.Fatal(err)
	}
	catalog, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	request := func(h http.Handler, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	h, err := NewHandler(s, Options{Languages: catalog, Now: func() time.Time { return time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	w := request(h, "/alpha/rewind?year=2025&tz=UTC")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Sua retrospectiva de 2025") || !strings.Contains(w.Body.String(), "Artista mais ouvido") || !strings.Contains(w.Body.String(), "Alpha Track") {
		t.Fatal("populated rewind not localized")
	}
	h, err = NewHandler(mediaBrokenBlocks{s}, Options{Languages: catalog})
	if err != nil {
		t.Fatal(err)
	}
	w = request(h, "/alpha/artist/1")
	for _, want := range []string{"Ranking indisponível", "Histórico de reproduções indisponível", "Álbuns indisponíveis"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("partial error missing: %s", want)
		}
	}
	if w.Code != 200 || strings.Contains(w.Body.String(), "private ") {
		t.Fatal("partial errors leaked or changed status")
	}
}

// TestPublicCatalogFallbackAndEscaping checks external strings in text and attribute contexts.
func TestPublicCatalogFallbackAndEscaping(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"en.json", "pt-BR.json"} {
		raw, err := os.ReadFile(filepath.Join("../../langs", name))
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err = json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		if name == "pt-BR.json" {
			delete(m, "ui.all_time_stats")
			m["ui.search"] = `Busca"><script>alert(1)</script>`
		}
		raw, err = json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := i18n.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(pageStore(t), Options{Languages: catalog})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/alpha", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "All time stats") || strings.Contains(w.Body.String(), "<script>alert(1)</script>") || !strings.Contains(w.Body.String(), "&lt;script&gt;") {
		t.Fatal("catalog fallback or HTML escaping failed")
	}
}
