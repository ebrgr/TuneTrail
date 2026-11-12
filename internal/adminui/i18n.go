package adminui

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/i18n"
)

// adminText formats only application-owned messages; template output remains escaped.
func adminText(locale *i18n.View, key, fallback string, args ...any) string {
	values := map[string]any{}
	for i := 0; i+1 < len(args); i += 2 {
		name, ok := args[i].(string)
		if ok {
			values[name] = args[i+1]
		}
	}
	return adminFormat(locale, key, fallback, values)
}

// adminFormat localizes complete messages with structured, non-secret values.
func adminFormat(locale *i18n.View, key, fallback string, values map[string]any) string {
	if locale != nil && key != "" {
		value := locale.Format(key, values, nil)
		if value != key {
			return value
		}
	}
	for name, value := range values {
		fallback = strings.ReplaceAll(fallback, "{{."+name+"}}", fmt.Sprint(value))
	}
	return fallback
}

// adminFuncs binds presentation messages to one request, never to a shared template.
func adminFuncs(locale *i18n.View) template.FuncMap {
	return template.FuncMap{
		"tr":  func(key, fallback string) string { return adminText(locale, key, fallback) },
		"trn": func(key, fallback string, args ...any) string { return adminText(locale, key, fallback, args...) },
		"trnp": func(key, fallback string, count int64, args ...any) string {
			values := map[string]any{"Count": count}
			for i := 0; i+1 < len(args); i += 2 {
				if name, ok := args[i].(string); ok {
					values[name] = args[i+1]
				}
			}
			if locale != nil {
				values["Count"] = locale.Number(count)
				plural := count
				if count == 0 {
					plural = 2
				}
				result := locale.Format(key, values, plural)
				if result != key {
					return result
				}
			}
			return adminFormat(nil, "", fallback, values)
		},
	}
}

// adminTemplate clones the immutable template before binding request-local language.
func (h *Handler) adminTemplate(locale *i18n.View) (*template.Template, error) {
	clone, err := h.tmpl.Clone()
	if err != nil {
		return nil, err
	}
	return clone.Funcs(adminFuncs(locale)), nil
}

// messageID maps only fixed server-owned diagnostics to translation keys.
func messageID(text string) string { return adminMessages[text] }
func localMessage(locale *i18n.View, text string) string {
	if key := messageID(text); key != "" {
		return adminText(locale, key, text)
	}
	return text
}

// textError keeps infrastructure failures localized without exposing internal errors.
func textError(w http.ResponseWriter, text string, status int) {
	http.Error(w, localMessage(i18n.ViewOf(w), text), status)
}
