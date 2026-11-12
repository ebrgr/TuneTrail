package i18n

import (
	"net/http/httptest"
	"testing"
	"time"
)

// TestPresentationLocale verifies plural selection and local display without changing instants.
func TestPresentationLocale(t *testing.T) {
	c, err := Load("../../langs")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		lang string
		n    int64
		want string
	}{
		{"en", 0, "0 plays"}, {"en", 1, "1 play"}, {"en", 2, "2 plays"},
		{"pt-BR", 0, "0 reproduções"}, {"pt-BR", 1, "1 reprodução"}, {"pt-BR", 2, "2 reproduções"},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Accept-Language", tc.lang)
		if got := c.Resolve(r).Quantity("quantity.plays", tc.n); got != tc.want {
			t.Errorf("%s %d: %q, want %q", tc.lang, tc.n, got, tc.want)
		}
	}
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2024, 3, 10, 1, 30, 0, 0, loc)
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "pt-BR")
	v := c.Resolve(r)
	if got := v.Date(date); got != "10/03/2024" {
		t.Fatal(got)
	}
	if date.Add(time.Hour).Hour() != 3 {
		t.Fatal("DST changed")
	}
	if got := v.Number(12345); got != "12.345" {
		t.Fatal(got)
	}
	if got := v.Decimal(12.5); got != "12,5" {
		t.Fatal(got)
	}
}
