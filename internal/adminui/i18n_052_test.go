package adminui

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db/sqlite"

	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/google/uuid"
)

// TestNoticeUsesLanguageAtRead preserves session isolation and resolves the redirected language.
func TestNoticeUsesLanguageAtRead(t *testing.T) {
	c, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	lang := func(value string) *i18n.View {
		r := httptest.NewRequest(http.MethodGet, "/admin", nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: value})
		return c.Resolve(r)
	}
	h := &Handler{notices: map[uuid.UUID]notice{}}
	first, second := uuid.New(), uuid.New()
	h.putNotice(first, 7, "Changes saved.")
	if v := h.takeNotice(second, 7, lang("pt-BR")); v != "" {
		t.Fatal("notice leaked across sessions")
	}
	if v := h.takeNotice(first, 8, lang("pt-BR")); v != "" {
		t.Fatal("notice leaked across profiles")
	}
	if v := h.takeNotice(first, 7, lang("pt-BR")); v != "Alterações salvas." {
		t.Fatalf("notice not localized at read: %q", v)
	}
	if v := h.takeNotice(first, 7, lang("en")); v != "" {
		t.Fatal("notice was not consumed")
	}
	h.putNoticeData(first, 7, "admin.import.summary", "Processed: {{.Processed}}. Imported: {{.Imported}}. Duplicates: {{.Duplicates}}. Ignored: {{.Ignored}}. Errors: {{.Errors}}.", map[string]any{"Processed": 2, "Imported": 1, "Duplicates": 1, "Ignored": 0, "Errors": 0})
	if v := h.takeNotice(first, 7, lang("pt-BR")); v != "Processadas: 2. Importadas: 1. Duplicadas: 1. Ignoradas: 0. Erros: 0." {
		t.Fatalf("import summary: %q", v)
	}
}

// TestLocalizedRateLimitKeepsHTMLAndStatus verifies the shared limiter response for HTML only.
func TestLocalizedRateLimitKeepsHTMLAndStatus(t *testing.T) {
	store, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(context.Background())
	c, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	limiter := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		})
	}
	h, err := New(store, Options{Languages: c, LoginMiddleware: limiter})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "http://example.test/admin/login", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 429 || !strings.Contains(w.Body.String(), "muitas tentativas") || !strings.Contains(w.Header().Get("Content-Type"), "text/html") || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("rate limit response %d %s", w.Code, w.Body.String())
	}
}

// TestLocalizedArgumentsRemainEscaped checks that names in translations cannot inject HTML.
func TestLocalizedArgumentsRemainEscaped(t *testing.T) {
	c, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "tunetrail_language", Value: "pt-BR"})
	tpl, err := template.New("test").Funcs(adminFuncs(c.Resolve(r))).Parse(`{{trn "admin.manage.recent" "Recent listens for {{.Name}}" "Name" .}}`)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, `<script>alert(1)</script>`); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "<script>") || !strings.Contains(b.String(), "&lt;script&gt;") {
		t.Fatalf("argument not escaped: %s", b.String())
	}
}
