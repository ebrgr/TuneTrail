package webui

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/i18n"
)

func loadTemplates() (*template.Template, error) {
	tmpl, err := template.New("page.html").Funcs(formatFuncs(nil)).Funcs(template.FuncMap{
		"tr": func(_ string, fallback string) string { return fallback },
		"tr2": func(_ string, fallback, name1 string, value1 any, name2 string, value2 any) string {
			return strings.NewReplacer("{{."+name1+"}}", fmt.Sprint(value1), "{{."+name2+"}}", fmt.Sprint(value2)).Replace(fallback)
		},
		"tr1": func(_ string, fallback string, name string, value any) string {
			return strings.ReplaceAll(fallback, "{{."+name+"}}", fmt.Sprint(value))
		},
		"avatarIcon": func(name string) string {
			return map[string]string{"music": "Music2", "headphones": "Headphones", "disc": "Disc3", "radio": "Radio", "user": "UserRound"}[name]
		},
		"icon": func(name string) template.HTML {
			data, err := files.ReadFile("static/icons/" + name + ".svg")
			if err != nil {
				return ""
			}
			return template.HTML(data)
		}, // Only bundled, reviewed SVGs; never catalog/user HTML.
		"tail": func(items []Media) []Media {
			if len(items) < 2 {
				return nil
			}
			return items[1:]
		},
		"first": func(items []Media) Media {
			if len(items) == 0 {
				return Media{}
			}
			return items[0]
		},
		"initials": func(s string) string {
			var letters []rune
			for _, word := range strings.Fields(s) {
				letters = append(letters, []rune(word)[0])
				if len(letters) == 2 {
					break
				}
			}
			return strings.ToUpper(string(letters))
		},
		"rank":         func(i int) string { return fmt.Sprintf("%02d", i+1) },
		"splitPeriods": func() []string { return []string{"day", "week", "month", "year", "all_time"} },
		"ago": func(i int) string {
			minutes := (i + 1) * 5
			if minutes >= 60 {
				hours := minutes / 60
				if hours == 1 {
					return "1 hour ago"
				}
				return fmt.Sprintf("%d hours ago", hours)
			}
			return fmt.Sprintf("%d minutes ago", minutes)
		},
		"history": func(items []Media) []Media {
			out := []Media{}
			if len(items) == 0 {
				return out
			}
			for i := 0; i < 28; i++ {
				out = append(out, items[i%len(items)])
			}
			return out
		},
	}).ParseFS(files, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// localizedTemplates binds translations to a private clone for each response.
func localizedTemplates(base *template.Template, locale *i18n.View) (*template.Template, error) {
	clone, err := base.Clone()
	if err != nil {
		return nil, err
	}
	clone.Funcs(formatFuncs(locale)).Funcs(template.FuncMap{
		"tr": func(key, fallback string) string {
			if locale == nil {
				return fallback
			}
			value := locale.T(key)
			if value == key {
				return fallback
			}
			return value
		},
		"tr2": func(key, fallback, name1 string, value1 any, name2 string, value2 any) string {
			if locale != nil {
				translated := locale.Format(key, map[string]any{name1: value1, name2: value2}, nil)
				if translated != key {
					return translated
				}
			}
			return strings.NewReplacer("{{."+name1+"}}", fmt.Sprint(value1), "{{."+name2+"}}", fmt.Sprint(value2)).Replace(fallback)
		},
		"tr1": func(key, fallback, name string, value any) string {
			if locale != nil {
				translated := locale.Format(key, map[string]any{name: value}, nil)
				if translated != key {
					return translated
				}
			}
			return strings.ReplaceAll(fallback, "{{."+name+"}}", fmt.Sprint(value))
		},
	})
	return clone, nil
}

// translated keeps UI labels separate from user-supplied names and identifiers.
func translated(locale *i18n.View, key, fallback string) string {
	if locale == nil {
		return fallback
	}
	text := locale.T(key)
	if text == key {
		return fallback
	}
	return text
}

// formatFuncs formats only visible UI content; raw dates and values remain unchanged.
func formatFuncs(locale *i18n.View) template.FuncMap {
	return template.FuncMap{
		"date":     func(t time.Time) string { return locale.Date(t) },
		"datetime": func(t time.Time) string { return locale.DateTime(t) },
		"weekday":  func(s string) string { return locale.Weekday(s) },
		"number":   func(n any) string { return locale.Number(n) },
		"decimal":  func(n float32) string { return locale.Decimal(float64(n)) },
		"unit": func(key string, n any) string {
			count, _ := strconv.ParseInt(fmt.Sprint(n), 10, 64)
			plural := count
			if count == 0 {
				plural = 2
			}
			if locale != nil {
				if value := locale.Format(key, nil, plural); value != key {
					return value
				}
			}
			name := strings.TrimPrefix(key, "unit.")
			if count == 1 {
				name = strings.TrimSuffix(name, "s")
			}
			return name
		},
		"quantity": func(key string, n any) string {
			count, _ := strconv.ParseInt(fmt.Sprint(n), 10, 64)
			if locale != nil {
				if value := locale.Quantity(key, count); value != key {
					return value
				}
			}
			word := map[string]string{"quantity.plays": "play", "quantity.minutes": "minute", "quantity.intervals": "interval"}[key]
			if count != 1 {
				word += "s"
			}
			return fmt.Sprintf("%d %s", count, word)
		},
	}
}
