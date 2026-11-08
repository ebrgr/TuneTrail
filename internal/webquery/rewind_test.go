package webquery

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
)

func TestRewindSelectionAndTimezone(t *testing.T) {
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 30, 0, 0, time.UTC)
	cases := []struct {
		query          string
		year, month    int
		progress, show bool
	}{
		{"tz=UTC", 2025, 0, false, false},
		{"tz=America/Sao_Paulo", 2025, 11, false, false},
		{"tz=UTC&year=2024", 2024, 0, false, false},
		{"tz=UTC&year=2024&month=2&show_time=1", 2024, 2, false, true},
		{"tz=UTC&year=2025&month=0", 2025, 0, false, false},
		{"tz=UTC&month=0", 2026, 0, true, false},
		{"tz=UTC&month=1", 2026, 1, true, false},
		{"tz=UTC&month=12", 2025, 12, false, false},
		{"tz=UTC&year=1&month=1", 1, 1, false, false},
		{"tz=UTC&year=2026&show_time=0", 2026, 0, true, false},
	}
	for _, tc := range cases {
		selection, err := Rewind(httptest.NewRequest("GET", "/?"+tc.query, nil), now)
		if err != nil || selection.Year != tc.year || selection.Month != tc.month || selection.InProgress != tc.progress || selection.ShowTime != tc.show {
			t.Fatalf("%s: %+v %v", tc.query, selection, err)
		}
		if selection.Timeframe.Resolved || selection.Timeframe.Year != tc.year || selection.Timeframe.Month != tc.month {
			t.Fatal("summary must resolve raw calendar once")
		}
		if tc.month == 0 && selection.Query.Has("month") {
			t.Fatal("annual month=0 was not normalized")
		}
	}
	for _, query := range []string{"year=0", "year=-1", "year=10000", "year=9999999999999999999999", "year=NaN", "year=undefined", "year=", "month=-1", "month=13", "month=undefined", "year=2027", "year=2026&month=2", "year=2025&year=2024", "month=1&month=2", "week=1", "from=1", "period=year", "artist_id=1", "show_time=2", "tz=UTC&tz=Asia/Tokyo"} {
		if _, err := Rewind(httptest.NewRequest("GET", "/?"+query, nil), now); err == nil {
			t.Fatalf("accepted %s", query)
		}
	}
	futureNow := time.Date(9999, 12, 31, 12, 0, 0, 0, time.UTC)
	if s, err := Rewind(httptest.NewRequest("GET", "/?year=9999&month=12&tz=UTC", nil), futureNow); err != nil || !s.InProgress {
		t.Fatal("year 9999 rejected")
	}
}
func TestRewindForcedTimezoneOverridesQueryAndCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/?tz=UTC", nil)
	req.Header.Set("Cookie", "tz=America/Sao_Paulo")
	forced, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	s, err := rewindAt(req, time.Date(2026, 1, 31, 16, 0, 0, 0, time.UTC), TimezoneWithForce(req, forced))
	if err != nil || s.Year != 2026 || s.Month != 1 || s.Timeframe.Timezone.String() != "Asia/Tokyo" {
		t.Fatalf("forced timezone: %+v %v", s, err)
	}
}
