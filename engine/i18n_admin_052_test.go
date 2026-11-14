package engine

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAdministrationLanguages checks localized login, protected pages and isolated concurrent requests.
func TestAdministrationLanguages(t *testing.T) {
	a := newProfileTestApp(t)
	get := func(path, language string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://koito.test"+path, nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: language})
		if authenticated {
			r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		}
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		return w
	}
	login := get("/admin", "pt-BR", false)
	require.Equal(t, 200, login.Code)
	require.Contains(t, login.Body.String(), "Nome de usuário")
	require.Contains(t, login.Body.String(), `name="username"`)
	require.Contains(t, login.Body.String(), "Senha")
	en, pt := get("/admin?profile=1", "en", true), get("/admin?profile=1", "pt-BR", true)
	require.Equal(t, 200, pt.Code)
	require.Contains(t, pt.Body.String(), "Importar e exportar")
	require.Contains(t, pt.Body.String(), "Salvar relay")
	require.Contains(t, pt.Body.String(), `aria-label="Navegação principal"`)
	require.Contains(t, pt.Body.String(), `data-tip="Sobre ↗"`)
	require.Contains(t, pt.Header().Get("Cache-Control"), "no-store")
	actions := regexp.MustCompile(`action="[^"]+"`)
	require.Equal(t, actions.FindAllString(en.Body.String(), -1), actions.FindAllString(pt.Body.String(), -1))
	config := get("/admin/settings", "pt-BR", true)
	require.Equal(t, 200, config.Code)
	require.Contains(t, config.Body.String(), "Nível de log")
	require.Contains(t, config.Body.String(), `name="KOITO_LOG_LEVEL"`)
	review := get("/admin/profile/1/delete-review", "pt-BR", true)
	require.Equal(t, 200, review.Code)
	require.Contains(t, review.Body.String(), "Excluir perfil")
	cross := httptest.NewRequest("POST", "http://koito.test/admin/login", strings.NewReader(""))
	cross.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
	cross.Header.Set("Origin", "http://evil.test")
	crossW := httptest.NewRecorder()
	a.router.ServeHTTP(crossW, cross)
	require.Equal(t, 403, crossW.Code)
	require.Contains(t, crossW.Body.String(), "solicitação não permitida")
	csrf := httptest.NewRequest("POST", "http://koito.test/admin/profile/1/save", strings.NewReader(""))
	csrf.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
	csrf.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	csrf.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	csrfW := httptest.NewRecorder()
	a.router.ServeHTTP(csrfW, csrf)
	require.Equal(t, 403, csrfW.Code)
	require.Contains(t, csrfW.Body.String(), "solicitação não permitida")
	// Save in English, then consume the one-use notice in Portuguese after redirect.
	profile, err := a.store.GetListeningProfileByID(context.Background(), 1)
	require.NoError(t, err)
	field := func(body, name string) string {
		m := regexp.MustCompile(`name="` + name + `" value="([^"]+)"`).FindStringSubmatch(body)
		require.Len(t, m, 2, name)
		return m[1]
	}
	values := url.Values{"csrf_token": {field(en.Body.String(), "csrf_token")}, "submission_token": {field(en.Body.String(), "submission_token")}, "profile_slug": {profile.Slug}, "display_name": {profile.DisplayName}, "slug": {profile.Slug}}
	post := httptest.NewRequest("POST", "http://koito.test/admin/profile/1/save", strings.NewReader(values.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "en"})
	post.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	saved := httptest.NewRecorder()
	a.router.ServeHTTP(saved, post)
	require.Equal(t, 303, saved.Code)
	notice := get("/admin?profile=1", "pt-BR", true)
	require.Contains(t, notice.Body.String(), "Alterações salvas.")
	jsonReq := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/login", strings.NewReader("{}"))
	jsonReq.Header.Set("Origin", "http://evil.test")
	jsonW := httptest.NewRecorder()
	a.router.ServeHTTP(jsonW, jsonReq)
	require.Equal(t, 403, jsonW.Code)
	require.Contains(t, jsonW.Body.String(), `"error"`)
	var wg sync.WaitGroup
	errs := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lang, want := "en", "Import and export"
			if i%2 == 0 {
				lang, want = "pt-BR", "Importar e exportar"
			}
			w := get("/admin?profile=1", lang, true)
			if w.Code != 200 || !strings.Contains(w.Body.String(), want) {
				errs <- fmt.Sprint(i)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("incorrect concurrent language %s", e)
	}
}
