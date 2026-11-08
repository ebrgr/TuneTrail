package webquery

import (
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTMLContracts(t *testing.T) {
	_ = cfg.Load(func(string) string { return "" }, "test")
	for _, query := range []string{"limit=-1", "limit=0", "limit=501", "limit=9999999999999999", "page=-1", "page=999999999999", "month=13", "year=0", "week=0", "year=2025&week=53", "month=1&week=1", "period=unknown", "artist_id=-1", "track_id=2147483648", "to=100", "from=200&to=100", "from=1&year=2026"} {
		if _, _, e := HTML(httptest.NewRequest("GET", "/?"+query, nil), false); e == nil {
			t.Errorf("accepted %s", query)
		}
	}
	o, q, e := HTML(httptest.NewRequest("GET", "/?page=0&year=2026&week=1&period=day&tz=America/Sao_Paulo", nil), false)
	if e != nil {
		t.Fatal(e)
	}
	from, to := db.TimeframeToTimeRange(o.Timeframe)
	if o.Page != 1 || q.Get("page") != "1" || o.Timeframe.Period != "" || q.Has("period") || from.Format("2006-01-02") != "2025-12-29" || to.Format("2006-01-02") != "2026-01-04" {
		t.Fatalf("ISO/page contract: %+v %v %v", o, from, to)
	}
	o, _, e = HTML(httptest.NewRequest("GET", "/", nil), false)
	if e != nil || o.Timeframe.Period != db.PeriodAllTime {
		t.Fatal("listing default")
	}
	o, _, e = HTML(httptest.NewRequest("GET", "/", nil), true)
	if e != nil || o.Timeframe.Period != db.PeriodWeek {
		t.Fatal("home default")
	}
	r := httptest.NewRequest("GET", "/?tz=UTC", nil)
	r.Header.Set("Cookie", "tz=America/Sao_Paulo")
	if Timezone(r) != time.UTC {
		t.Fatal("query must override cookie")
	}
	r = httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Cookie", "tz=America/Sao_Paulo")
	if Timezone(r).String() != "America/Sao_Paulo" {
		t.Fatal("cookie timezone")
	}
	r = httptest.NewRequest("GET", "/?tz=UTC", nil)
	r.Header.Set("Cookie", "tz=America/Sao_Paulo")
	forced, err := time.LoadLocation("Asia/Tokyo")
	if err != nil || TimezoneWithForce(r, forced) != forced {
		t.Fatal("forced timezone must override query and cookie")
	}
}
func TestCalendarBoundariesAndRollingDay(t *testing.T) {
	for _, v := range []struct {
		year, month int
		start, end  string
	}{{2024, 2, "2024-02-01", "2024-02-29"}, {2026, 2, "2026-02-01", "2026-02-28"}, {2026, 12, "2026-12-01", "2026-12-31"}} {
		a, b := db.TimeframeToTimeRange(db.Timeframe{Year: v.year, Month: v.month, Timezone: time.UTC})
		if a.Format("2006-01-02") != v.start || b.Format("2006-01-02") != v.end {
			t.Fatal("month bounds")
		}
	}
	a, b := db.TimeframeToTimeRange(db.Timeframe{Period: db.PeriodDay, Timezone: time.UTC})
	if d := b.Sub(a); d < 23*time.Hour || d > 25*time.Hour {
		t.Fatal("day must remain a rolling interval")
	}
	loc, e := time.LoadLocation("America/New_York")
	if e != nil {
		t.Fatal(e)
	}
	a, b = db.TimeframeToTimeRange(db.Timeframe{Year: 2026, Month: 3, Timezone: loc})
	if a.UTC().Hour() != 5 || b.UTC().Hour() != 3 {
		t.Fatal("DST calendar boundary lost")
	}
}

func TestResolvedRollingBoundsMatchLabel(t *testing.T) {
	tf := ResolveTimeframe(db.Timeframe{Period: db.PeriodDay, Timezone: time.UTC})
	if tf.From.IsZero() || tf.To.IsZero() {
		t.Fatal("rolling bounds were not fixed")
	}
	a, b := db.TimeframeToTimeRange(tf)
	if !a.Equal(tf.From) || !b.Equal(tf.To) {
		t.Fatal("resolved interval changed")
	}
	if RangeLabel(tf) != a.In(time.UTC).Format("02 Jan 2006 15:04")+" – "+b.In(time.UTC).Format("02 Jan 2006 15:04") {
		t.Fatal("label differs from read interval")
	}
}

// TestTimezoneFallbackKeepsProcessLocalAndUsesValidCookieAfterInvalidQuery checks precedence.
func TestTimezoneFallbackKeepsProcessLocalAndUsesValidCookieAfterInvalidQuery(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("Local", -3*3600)
	defer func() { time.Local = original }()
	r := httptest.NewRequest("GET", "/?tz=Invalid%2FZone", nil)
	if got := TimezoneWithForce(r, nil); got != time.Local {
		t.Fatalf("process timezone lost: %s", got)
	}
	r.Header.Set("Cookie", "tz=Asia/Kathmandu")
	if got := TimezoneWithForce(r, nil); got.String() != "Asia/Kathmandu" {
		t.Fatalf("cookie fallback lost: %s", got)
	}
	r = httptest.NewRequest("GET", "/?tz=Asia%2FKatmandu", nil)
	if got := TimezoneWithForce(r, nil); got.String() != "Asia/Kathmandu" {
		t.Fatalf("alias not resolved: %s", got)
	}
}
