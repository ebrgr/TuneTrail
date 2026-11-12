// Package i18n loads external message catalogs and resolves language per request.
package i18n

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"text/template"

	"github.com/ebrgr/TuneTrail/internal/adminweb"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const cookieName = "tunetrail_language"

var tags = []language.Tag{language.English, language.BrazilianPortuguese}
var files = []string{"en.json", "pt-BR.json"}
var placeholder = regexp.MustCompile(`{{\s*\.([A-Za-z][A-Za-z0-9_]*)\s*}}`)

// Catalog contains immutable messages loaded from a directory outside the binary.
type Catalog struct {
	bundle  *goi18n.Bundle
	matcher language.Matcher
}

// View holds request-specific language and safe form return information.
type View struct {
	Tag, Return, Choice string
	localizer, fallback *goi18n.Localizer
}

// T returns ordinary text; html/template applies contextual escaping at the call site.
func (v *View) T(key string) string {
	if v == nil {
		return key
	}
	return v.Format(key, nil, nil)
}

// Format resolves named variables and CLDR plural counts while retaining English fallback.
func (v *View) Format(key string, data any, count any) string {
	if v == nil {
		return key
	}
	config := &goi18n.LocalizeConfig{MessageID: key, TemplateData: data, PluralCount: count}
	text, err := v.localizer.Localize(config)
	if err != nil {
		text, err = v.fallback.Localize(config)
	}
	if err != nil {
		return key
	}
	return text
}

// placeholders limits message expressions to named data fields, which makes catalogs comparable.
func placeholders(value string) ([]string, error) {
	matches := placeholder.FindAllStringSubmatch(value, -1)
	remainder := placeholder.ReplaceAllString(value, "")
	if strings.Contains(remainder, "{{") || strings.Contains(remainder, "}}") {
		return nil, fmt.Errorf("unsupported message expression")
	}
	names := map[string]bool{}
	for _, m := range matches {
		names[m[1]] = true
	}
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

// Directory selects the external catalogs, with an explicit override for installed binaries.
func Directory() (string, error) {
	if dir := os.Getenv("TUNETRAIL_LANGS_DIR"); dir != "" {
		return filepath.Abs(dir)
	}
	exe, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(exe), "langs")
		if st, e := os.Stat(p); e == nil && st.IsDir() {
			return p, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	// Find the checkout only when a module root accompanies the language directory.
	for dir := cwd; ; dir = filepath.Dir(dir) {
		p := filepath.Join(dir, "langs")
		if _, e := os.Stat(filepath.Join(dir, "go.mod")); e == nil {
			if st, e := os.Stat(p); e == nil && st.IsDir() {
				return p, nil
			}
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	if exe == "" {
		return "", fmt.Errorf("cannot locate executable or language catalogs")
	}
	return filepath.Join(filepath.Dir(exe), "langs"), nil
}

// Load validates supported catalogs and their message keys before serving requests.
func Load(dir string) (*Catalog, error) {
	b := goi18n.NewBundle(language.English)
	var keys map[string]bool
	var fields map[string][]string
	for i, name := range files {
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("invalid language catalog %s", path)
		}
		// Nonblocking open and no symlink following also protect against replacement races.
		f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
		if err != nil {
			return nil, fmt.Errorf("language catalog %s: %w", path, err)
		}
		info, err = f.Stat()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
			f.Close()
			return nil, fmt.Errorf("invalid language catalog %s", path)
		}
		data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
		f.Close()
		if err != nil || len(data) > 1<<20 {
			return nil, fmt.Errorf("cannot read language catalog %s", path)
		}
		var msgs map[string]json.RawMessage
		if err = json.Unmarshal(data, &msgs); err != nil || len(msgs) == 0 {
			return nil, fmt.Errorf("invalid language catalog %s", path)
		}
		found := map[string]bool{}
		foundFields := map[string][]string{}
		for key, raw := range msgs {
			if key == "" {
				return nil, fmt.Errorf("empty language message in %s", path)
			}
			forms := map[string]string{}
			var single string
			if json.Unmarshal(raw, &single) == nil {
				forms["other"] = single
			} else if err = json.Unmarshal(raw, &forms); err != nil {
				return nil, fmt.Errorf("invalid language message %s in %s", key, path)
			}
			if forms["other"] == "" {
				return nil, fmt.Errorf("missing other form %s in %s", key, path)
			}
			message := &goi18n.Message{ID: key}
			var args []string
			for form, value := range forms {
				if value == "" {
					return nil, fmt.Errorf("empty language message %s in %s", key, path)
				}
				if _, err = template.New(key).Parse(value); err != nil {
					return nil, fmt.Errorf("invalid language message %s in %s: %w", key, path, err)
				}
				fields, e := placeholders(value)
				if e != nil {
					return nil, fmt.Errorf("invalid language message %s in %s: %w", key, path, e)
				}
				if args == nil {
					args = fields
				} else if strings.Join(args, "\x00") != strings.Join(fields, "\x00") {
					return nil, fmt.Errorf("inconsistent plural placeholders %s in %s", key, path)
				}
				switch form {
				case "zero":
					message.Zero = value
				case "one":
					message.One = value
				case "two":
					message.Two = value
				case "few":
					message.Few = value
				case "many":
					message.Many = value
				case "other":
					message.Other = value
				default:
					return nil, fmt.Errorf("invalid plural form %s in %s", key, path)
				}
			}
			foundFields[key] = args
			found[key] = true
			if err = b.AddMessages(tags[i], message); err != nil {
				return nil, err
			}
		}
		if i == 0 {
			keys, fields = found, foundFields
		} else {
			for k := range found {
				if !keys[k] || strings.Join(fields[k], "\x00") != strings.Join(foundFields[k], "\x00") {
					return nil, fmt.Errorf("invalid key or placeholders %s in %s", k, path)
				}
			}
		}
	}
	for _, key := range []string{"language.label", "language.auto", "language.save", "profiles.title", "profiles.intro", "profiles.empty", "profiles.empty_help", "nav.administration", "settings.title"} {
		if !keys[key] {
			return nil, fmt.Errorf("language catalog missing key %s", key)
		}
	}
	return &Catalog{bundle: b, matcher: language.NewMatcher(tags)}, nil
}

// Resolve prefers an explicit browser cookie and otherwise negotiates Accept-Language.
func (c *Catalog) Resolve(r *http.Request) *View {
	if c == nil {
		return nil
	}
	choice := "auto"
	request := ""
	if cookie, err := r.Cookie(cookieName); err == nil && (cookie.Value == "en" || cookie.Value == "pt-BR") {
		choice = cookie.Value
		request = choice
	}
	if request == "" && len(r.Header.Get("Accept-Language")) <= 1024 {
		request = r.Header.Get("Accept-Language")
	}
	index := 0
	if parsed, _, err := language.ParseAcceptLanguage(request); err == nil && len(parsed) > 0 {
		_, index, _ = c.matcher.Match(parsed...)
	}
	v := &View{Tag: tags[index].String(), Return: adminweb.SafeReturn(r.URL.RequestURI()), Choice: choice}
	if v.Return == "" {
		v.Return = "/"
	}
	v.localizer = goi18n.NewLocalizer(c.bundle, v.Tag)
	v.fallback = goi18n.NewLocalizer(c.bundle, "en")
	return v
}

// Writer carries only request-local view data to existing buffered HTML renderers.
type Writer struct {
	http.ResponseWriter
	View *View
}

func (w *Writer) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// ViewOf returns the localizer attached to this response, if enabled.
func ViewOf(w http.ResponseWriter) *View {
	for w != nil {
		if v, ok := w.(*Writer); ok {
			return v.View
		}
		next, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			break
		}
		w = next.Unwrap()
	}
	return nil
}

// SetLanguage handles a same-origin POST without requiring JavaScript or authentication.
func (c *Catalog) SetLanguage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if len(r.PostForm) != 2 || len(r.PostForm["language"]) != 1 || len(r.PostForm["return_to"]) != 1 {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	choice := r.PostForm.Get("language")
	if choice != "auto" && choice != "en" && choice != "pt-BR" {
		http.Error(w, "invalid language", http.StatusBadRequest)
		return
	}
	next := adminweb.SafeReturn(r.PostForm.Get("return_to"))
	if next == "" {
		http.Error(w, "invalid return", http.StatusBadRequest)
		return
	}
	cookie := &http.Cookie{Name: cookieName, Value: choice, Path: "/", HttpOnly: true, Secure: adminweb.SecureRequest(r), SameSite: http.SameSiteLaxMode, MaxAge: 31536000}
	if choice == "auto" {
		cookie.Value = ""
		cookie.MaxAge = -1
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

// Wrap associates a locale with one response without changing shared templates.
func (c *Catalog) Wrap(w http.ResponseWriter, r *http.Request) http.ResponseWriter {
	if c == nil {
		return w
	}
	v := c.Resolve(r)
	w.Header().Set("Content-Language", v.Tag)
	w.Header().Add("Vary", "Accept-Language")
	return &Writer{ResponseWriter: w, View: v}
}
