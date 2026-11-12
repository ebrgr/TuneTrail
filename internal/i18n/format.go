package i18n

import (
	"fmt"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Number formats visible numbers without altering machine-readable attributes.
func (v *View) Number(n any) string {
	tag := language.English
	if v != nil && v.Tag == "pt-BR" {
		tag = language.BrazilianPortuguese
	}
	return message.NewPrinter(tag).Sprint(n)
}

// Decimal formats a visible value with one fractional digit.
func (v *View) Decimal(n float64) string {
	tag := language.English
	if v != nil && v.Tag == "pt-BR" {
		tag = language.BrazilianPortuguese
	}
	return message.NewPrinter(tag).Sprintf("%.1f", n)
}

// Date formats a civil date in its existing location; it does not change the instant.
func (v *View) Date(t time.Time) string {
	if v != nil && v.Tag == "pt-BR" {
		return t.Format("02/01/2006")
	}
	return t.Format("01/02/2006")
}

// DateTime renders a local date and time without affecting the supplied timezone.
func (v *View) DateTime(t time.Time) string { return v.Date(t) + t.Format(" 15:04:05 MST") }

// Month returns catalog names, keeping the numeric month independent from presentation.
func (v *View) Month(m time.Month, short bool) string {
	key := fmt.Sprintf("date.month.%d", m)
	if short {
		key += ".short"
	}
	if v != nil {
		if name := v.T(key); name != key {
			return name
		}
	}
	name := m.String()
	if short && len(name) >= 3 {
		return name[:3]
	}
	return name
}

// Weekday localizes existing calendar labels without changing the first weekday.
func (v *View) Weekday(name string) string {
	if name == "" {
		return ""
	}
	if v != nil {
		if text := v.T("date.weekday." + name); text != "date.weekday."+name {
			return text
		}
	}
	return name
}

// Quantity resolves a complete CLDR message using the raw count for plural selection.
func (v *View) Quantity(key string, count int64) string {
	if v == nil {
		return fmt.Sprint(count)
	}
	// Portuguese CLDR classifies zero as singular; the UI uses the conventional plural for zero.
	pluralCount := count
	if count == 0 {
		pluralCount = 2
	}
	return v.Format(key, map[string]any{"Count": v.Number(count)}, pluralCount)
}
