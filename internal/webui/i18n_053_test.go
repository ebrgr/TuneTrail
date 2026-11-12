package webui

import (
	"bytes"
	"github.com/ebrgr/TuneTrail/internal/db"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/i18n"
)

// TestLocalizedRelativeTime checks thresholds without modifying the original clock or timezone.
func TestLocalizedRelativeTime(t *testing.T) {
	c, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "pt-BR")
	v := c.Resolve(r)
	now := time.Date(2024, 12, 31, 23, 59, 0, 0, time.FixedZone("BRT", -3*3600))
	for _, tc := range []struct {
		d    time.Duration
		want string
	}{{0, "agora mesmo"}, {time.Minute, "1 minuto atrás"}, {2 * time.Minute, "2 minutos atrás"}, {time.Hour, "1 hora atrás"}, {2 * time.Hour, "2 horas atrás"}, {24 * time.Hour, "30/12/2024 23:59"}} {
		if got := localizedRelativeTime(v, now, now.Add(-tc.d)); got != tc.want {
			t.Errorf("%s: %q want %q", tc.d, got, tc.want)
		}
	}
}

// TestGraphLocalizationPreservesMachineData checks SVG geometry and timestamps across languages.
func TestGraphLocalizationPreservesMachineData(t *testing.T) {
	c, err := i18n.Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	base, err := loadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	from := time.Date(2024, 3, 10, 1, 30, 0, 0, time.FixedZone("EST", -5*3600))
	to := from.Add(time.Hour)
	graph, err := buildInterest([]db.InterestBucket{{BucketStart: from, BucketEnd: to, ListenCount: 1}}, from.Location(), "test")
	if err != nil {
		t.Fatal(err)
	}
	render := func(lang string) string {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", lang)
		tpl, e := localizedTemplates(base, c.Resolve(r))
		if e != nil {
			t.Fatal(e)
		}
		var out bytes.Buffer
		if e = tpl.ExecuteTemplate(&out, "interest-graph", graph); e != nil {
			t.Fatal(e)
		}
		return out.String()
	}
	en, pt := render("en"), render("pt-BR")
	for _, value := range []string{graph.Line, graph.Area, from.Format("2006-01-02T15:04:05Z07:00"), to.Format("2006-01-02T15:04:05Z07:00")} {
		if !strings.Contains(en, value) || !strings.Contains(pt, value) {
			t.Errorf("machine value changed: %s", value)
		}
	}
	if !strings.Contains(en, "1 play in 1 interval") || !strings.Contains(pt, "1 reprodução em 1 intervalo") {
		t.Fatal("accessible description not localized")
	}
	if !strings.Contains(en, "03/10/2024") || !strings.Contains(pt, "10/03/2024") {
		t.Fatal("dates not localized")
	}
}
