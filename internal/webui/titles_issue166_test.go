package webui

// ISSUE-166: page tabs follow one pattern — the home tab carries the profile
// name and every other public tab carries its translated page name, always
// with the " - TuneTrail" suffix appended by the shared head template. The
// Koito brand remains only in the footer attribution and the import format.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/ebrgr/TuneTrail/internal/webassets"
)

func TestPageTitlesFollowLanguageAndProfile(t *testing.T) {
	s := pageStore(t)
	// Give the beta profile one 2025 listen so its rewind has content to title.
	testfixtures.OccurrenceListen(t, s, db.SaveListenOpts{ProfileID: 2, TrackID: 1, Time: time.Date(2025, 6, 10, 12, 0, 0, 0, time.UTC)})
	catalog, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(s, Options{Languages: catalog, Now: func() time.Time { return time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	title := func(lang, path string) string {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: lang})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s %s: status %d", lang, path, w.Code)
		}
		body := w.Body.String()
		start := strings.Index(body, "<title>")
		end := strings.Index(body, "</title>")
		if start < 0 || end <= start {
			t.Fatalf("%s %s: missing <title>", lang, path)
		}
		return body[start+len("<title>") : end]
	}
	cases := []struct {
		path, en, ptBR string
	}{
		// The home tab carries the profile name; the directory and pages use translated labels.
		{"/beta", "Beta - TuneTrail", "Beta - TuneTrail"},
		{"/", "Listening profiles - TuneTrail", "Perfis de reprodução - TuneTrail"},
		{"/beta/listens?period=all_time", "Last Played - TuneTrail", "Últimas reproduções - TuneTrail"},
		{"/beta/chart/top/artists?period=all_time", "Top Artists - TuneTrail", "Artistas mais ouvidos - TuneTrail"},
		{"/beta/chart/top/albums?period=all_time", "Top Albums - TuneTrail", "Álbuns mais ouvidos - TuneTrail"},
		{"/beta/chart/top/tracks?period=all_time", "Top Tracks - TuneTrail", "Faixas mais ouvidas - TuneTrail"},
		{"/beta/search?q=track", "Search - TuneTrail", "Buscar - TuneTrail"},
		{"/beta/rewind?year=2025&tz=UTC", "Your 2025 Rewind - TuneTrail", "Sua retrospectiva de 2025 - TuneTrail"},
		{"/beta/rewind?year=2025&month=6&tz=UTC", "Your June 2025 Rewind - TuneTrail", "Sua retrospectiva de junho 2025 - TuneTrail"},
		// Media keeps the media name in both languages; only the suffix is applied.
		{"/beta/track/1", "Alpha Track - TuneTrail", "Alpha Track - TuneTrail"},
	}
	for _, tc := range cases {
		if got := title("en", tc.path); got != tc.en {
			t.Errorf("en %s title = %q, want %q", tc.path, got, tc.en)
		}
		if got := title("pt-BR", tc.path); got != tc.ptBR {
			t.Errorf("pt-BR %s title = %q, want %q", tc.path, got, tc.ptBR)
		}
	}
	// No public tab may carry the Koito brand in its title.
	for _, path := range []string{"/", "/beta", "/beta/listens?period=all_time", "/beta/chart/top/artists?period=all_time", "/beta/chart/top/albums?period=all_time", "/beta/chart/top/tracks?period=all_time", "/beta/search?q=x", "/beta/rewind?year=2025&tz=UTC", "/beta/track/1"} {
		for _, lang := range []string{"en", "pt-BR"} {
			if got := title(lang, path); strings.Contains(got, "Koito") {
				t.Errorf("%s %s title contains Koito: %q", lang, path, got)
			}
		}
	}
}

func TestErrorTitlesAndKoitoAttributionRemain(t *testing.T) {
	s := pageStore(t)
	catalog, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(s, Options{Languages: catalog})
	if err != nil {
		t.Fatal(err)
	}
	// Error documents translate their status label under the same suffix pattern.
	for _, tc := range []struct{ lang, want string }{
		{"en", "Not Found - TuneTrail"},
		{"pt-BR", "Não encontrado - TuneTrail"},
	} {
		r := httptest.NewRequest("GET", "/missing", nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: tc.lang})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 404 || !strings.Contains(w.Body.String(), "<title>"+tc.want+"</title>") {
			t.Fatalf("%s 404 title: got %d %q", tc.lang, w.Code, w.Body.String())
		}
	}
	// The footer attribution keeps the Koito brand on live pages.
	r := httptest.NewRequest("GET", "/beta", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "en"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Koito 0.3.2") || !strings.Contains(w.Body.String(), "koito-original.svg") {
		t.Fatal("footer attribution lost")
	}
	// The Koito import format option remains in the administration template.
	raw, err := webassets.FS{Prefix: "admin"}.ReadFile("templates/account-tabs.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `<option value="koito">Koito</option>`) {
		t.Fatal("Koito import option lost")
	}
}
