package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestLanguageJourney checks localized fallbacks regardless of markup indentation, origin and isolation.
func TestLanguageJourney(t *testing.T) {
	a := newProfileTestApp(t)
	page := func(path, locale string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://koito.test"+path, nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: locale})
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		return w
	}
	pt := page("/", "pt-BR")
	if pt.Code != 200 || !strings.Contains(pt.Body.String(), "Escolha um perfil de reprodução") || pt.Header().Get("Content-Language") != "pt-BR" {
		t.Fatal("Portuguese page")
	}
	en := page("/", "en")
	if !strings.Contains(en.Body.String(), "Choose a listening profile") {
		t.Fatal("English page")
	}
	// ISSUE-110: the language form belongs to the theme panel, a native popover that also
	// opens without JavaScript, so there is no separate no-JS copy of the form.
	checkSettings := func(body string) {
		t.Helper()
		start := strings.Index(body, `id="theme-panel"`)
		if start < 0 || !strings.Contains(body, `popovertarget="theme-panel"`) {
			t.Fatal("theme panel or its trigger missing")
		}
		if !strings.Contains(body[start:], `name="language"`) {
			t.Fatal("language form outside theme panel")
		}
		if strings.Count(body, `name="language"`) != 1 {
			t.Fatal("unexpected language selector on page")
		}
		if strings.Contains(body, "settings-dialog") || strings.Contains(body, "admin-settings") {
			t.Fatal("old settings dialog still rendered")
		}
	}
	checkSettings(pt.Body.String())
	admin := page("/admin", "pt-BR")
	if admin.Code != 200 || !strings.Contains(admin.Body.String(), "Aplicar idioma") {
		t.Fatalf("admin language selector: %d", admin.Code)
	}
	checkSettings(admin.Body.String())
	req := httptest.NewRequest("POST", "http://koito.test/language", strings.NewReader("language=pt-BR&return_to=%2Fadmin"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://evil.test")
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatalf("cross origin: %d", w.Code)
	}
	var wg sync.WaitGroup
	errs := make(chan string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			locale, want := "en", "Choose a listening profile"
			if i%2 == 0 {
				locale, want = "pt-BR", "Escolha um perfil de reprodução"
			}
			res := page("/", locale)
			if !strings.Contains(res.Body.String(), want) {
				errs <- fmt.Sprint(i)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("request %s mixed locale", e)
	}
}
