package engine

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestPublicTranslationsKeepMediaAndProtocolStable exercises nested templates in two languages.
func TestPublicTranslationsKeepMediaAndProtocolStable(t *testing.T) {
	a := newProfileTestApp(t)
	for _, q := range []string{
		`INSERT INTO artists(id) VALUES (40)`,
		`INSERT INTO artist_aliases VALUES (40,'Top albums','User',1)`,
		`INSERT INTO releases(id,various_artists) VALUES (40,0)`,
		`INSERT INTO release_aliases VALUES (40,'Shared Album','User',1)`,
		`INSERT INTO artist_releases VALUES (40,40,1)`,
		`INSERT INTO tracks(id,release_id,duration) VALUES (40,40,180)`,
		`INSERT INTO track_aliases VALUES (40,'Top tracks','User',1)`,
		`INSERT INTO artist_tracks VALUES (40,40,1)`,
	} {
		require.NoError(t, a.store.Exec(q))
	}
	require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,40,?)`, time.Now().Add(-time.Hour).Unix()))
	get := func(path, lang string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://koito.test"+path, nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: lang})
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		return w
	}
	for _, tc := range []struct{ path, pt string }{
		{"/default", "Estatísticas de todo o período"},
		{"/default/chart/top/tracks?period=all_time", "Faixas mais ouvidas"},
		{"/default/listens?period=all_time", "Últimas reproduções"},
		{"/default/search?q=Shared", "Buscar"},
		{"/default/artist/40", "Artista"},
		{"/default/album/40", "Álbum"},
		{"/default/track/40", "Faixa"},
		{"/default/rewind?year=2025", "retrospectiva"},
	} {
		en, pt := get(tc.path, "en"), get(tc.path, "pt-BR")
		require.Equal(t, 200, en.Code, tc.path)
		require.Equal(t, 200, pt.Code, tc.path)
		require.Contains(t, pt.Body.String(), tc.pt, tc.path)
		hrefs := regexp.MustCompile(`href="[^"]+"`)
		require.Equal(t, hrefs.FindAllString(en.Body.String(), -1), hrefs.FindAllString(pt.Body.String(), -1), tc.path)
	}
	home := get("/default", "pt-BR").Body.String()
	require.Contains(t, home, ">Top tracks</a>")
	require.Contains(t, home, ">Top albums</a>")
	require.Contains(t, home, "Faixas mais ouvidas")
	require.Contains(t, home, "data-i18n-delete-confirm=\"Excluir esta reprodução?\"")
	missing := get("/default/track/99999", "pt-BR")
	require.Equal(t, 404, missing.Code)
	require.Contains(t, missing.Body.String(), "Não encontrado")
	// Translated text and names remain escaped even inside localized sentence parameters.
	require.NoError(t, a.store.Exec(`UPDATE artist_aliases SET alias='<script>alert(1)</script>' WHERE artist_id=40`))
	escaped := get("/default/artist/40", "pt-BR").Body.String()
	require.Contains(t, escaped, "&lt;script&gt;alert(1)&lt;/script&gt;")
	require.NotContains(t, escaped, "<script>alert(1)</script>")
	var wg sync.WaitGroup
	failures := make(chan string, 20)
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			language, want := "en", "All time stats"
			if n%2 == 0 {
				language, want = "pt-BR", "Estatísticas de todo o período"
			}
			page := get("/default", language)
			if !strings.Contains(page.Body.String(), want) {
				failures <- language
			}
		}(n)
	}
	wg.Wait()
	close(failures)
	for failure := range failures {
		t.Errorf("mixed language: %s", failure)
	}
}
