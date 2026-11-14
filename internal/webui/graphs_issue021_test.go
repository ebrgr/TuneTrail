package webui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestRenderedGraphTablesMatchIndependentProfileCounts checks accessible values independently of HTML indentation.
func TestRenderedGraphTablesMatchIndependentProfileCounts(t *testing.T) {
	store := pageStore(t)
	handler, err := NewHandler(store, Options{})
	if err != nil {
		t.Fatal(err)
	}
	rowCount := regexp.MustCompile(`<td>([0-9]+)</td>\s*</tr>`)
	sumTable := func(t *testing.T, body, title string) int {
		t.Helper()
		marker := `<summary>` + title + `</summary>`
		start := strings.Index(body, marker)
		if start < 0 {
			t.Fatalf("missing %s", title)
		}
		end := strings.Index(body[start:], `</details>`)
		if end < 0 {
			t.Fatalf("unterminated %s", title)
		}
		matches := rowCount.FindAllStringSubmatch(body[start:start+end], -1)
		if len(matches) == 0 {
			t.Fatalf("no rows in %s", title)
		}
		total := 0
		for _, match := range matches {
			count, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			total += count
		}
		return total
	}
	for _, profile := range []struct {
		slug            string
		activity, track int
	}{
		{"alpha", 3, 2}, {"beta", 2, 1}, {"empty", 0, 0},
	} {
		t.Run(profile.slug, func(t *testing.T) {
			home := pageRequest(handler, "/"+profile.slug+"?period=all_time&tz=UTC")
			if home.Code != 200 {
				t.Fatal(home.Code)
			}
			if got := sumTable(t, home.Body.String(), "Activity values"); got != profile.activity {
				t.Fatalf("activity got %d, expected %d", got, profile.activity)
			}
			if profile.slug == "empty" {
				return
			} // An unheard track has no public media page.
			for _, kind := range []string{"artist", "album", "track"} {
				media := pageRequest(handler, fmt.Sprintf("/%s/%s/1?tz=UTC", profile.slug, kind))
				if media.Code != 200 {
					t.Fatal(media.Code)
				}
				expected := profile.activity
				if kind == "track" || profile.slug == "beta" {
					expected = profile.track
				}
				for _, title := range []string{"Activity values", "Interest values"} {
					if got := sumTable(t, media.Body.String(), title); got != expected {
						t.Fatalf("%s %s got %d, expected %d", kind, title, got, expected)
					}
				}
			}
		})
	}
}
