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

// Message is the data of the single message component (ISSUE-128). Kind is "error" (role=alert),
// "warning" or "success" (role=status); Section marks a message announced by an enhanced card update.
type Message struct {
	Kind, ID, Text string
	Section        bool
}

// newMessage builds a Message for the admin/message template; an optional fourth argument "section"
// sets Section. It is pure, so it is bound once at parse time.
func newMessage(kind, id, text string, options ...string) Message {
	m := Message{Kind: kind, ID: id, Text: text}
	for _, o := range options {
		m.Section = m.Section || o == "section"
	}
	return m
}

// Heading is one heading of the review cards (ISSUE-128).
type Heading struct {
	Level    int
	ID, Text string
}

// newHeading returns a card heading. A dialog fragment sits under its own h2 title, so every level
// shifts down by one; offset 1 is a sub-section of a card.
func newHeading(dialog bool, offset int, id, text string) Heading {
	level := 2 + offset
	if dialog {
		level++
	}
	return Heading{Level: level, ID: id, Text: text}
}

// statusTitle names a refused request without saying why it was refused: a 404 reads the same for a
// missing and for a hidden resource (ISSUE-128).
func statusTitle(locale *i18n.View, status int) string {
	switch status {
	case http.StatusUnauthorized:
		return adminText(locale, "admin.error.title_unauthorized", "Sign in required")
	case http.StatusForbidden:
		return adminText(locale, "admin.error.title_forbidden", "Access denied")
	case http.StatusNotFound:
		return adminText(locale, "admin.error.title_unavailable", "Page unavailable")
	}
	return adminText(locale, "ui.request_status", "Request status")
}

// adminFuncs binds presentation messages to one request, never to a shared template.
func adminFuncs(locale *i18n.View) template.FuncMap {
	return template.FuncMap{
		"statusTitle": func(status int) string { return statusTitle(locale, status) },
		"tr":          func(key, fallback string) string { return adminText(locale, key, fallback) },
		"trn":         func(key, fallback string, args ...any) string { return adminText(locale, key, fallback, args...) },
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
