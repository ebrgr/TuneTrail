package webui

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

func TestSpeculation075PublicDocumentsOnly(t *testing.T) {
	// Render the real document so escaping, session exclusion and cache headers are covered together.
	h, err := NewHandler(pageStore(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, cookie string
		want         bool
	}{
		{"/alpha", "", true}, {"/beta/track/1", "", true},
		{"/alpha", "tunetrail_session=invalid", false},
		{"/alpha", "tunetrail_session=", false},
		{"/", "", false}, {"/missing", "", false},
	} {
		r := httptest.NewRequest("GET", tc.path, nil)
		if tc.cookie != "" {
			r.Header.Set("Cookie", tc.cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		body := w.Body.String()
		start := strings.Index(body, `<script type="speculationrules">`)
		if (start >= 0) != tc.want {
			t.Fatalf("%s cookie=%q: speculation=%v", tc.path, tc.cookie, start >= 0)
		}
		if !tc.want {
			continue
		}
		if w.Header().Get("Cache-Control") != "private, no-cache" || w.Header().Get("Vary") != "Cookie" {
			t.Fatal("document cache policy changed")
		}
		text := strings.SplitN(body[start+len(`<script type="speculationrules">`):], "</script>", 2)[0]
		var rules struct {
			Prefetch []struct {
				Source, Eagerness string
				Where             struct {
					And []struct {
						Href []struct{ Pathname, Search string } `json:"href_matches"`
						Not  struct {
							Selector string `json:"selector_matches"`
						} `json:"not"`
					}
				}
			}
		}
		if err := json.Unmarshal([]byte(text), &rules); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, text)
		}
		if len(rules.Prefetch) != 1 || rules.Prefetch[0].Source != "document" || rules.Prefetch[0].Eagerness != "moderate" || strings.Contains(text, "prerender") {
			t.Fatal(text)
		}
		clauses := rules.Prefetch[0].Where.And
		slug := strings.Split(strings.TrimPrefix(tc.path, "/"), "/")[0]
		expected := []string{"/" + slug, "/" + slug + "/listens", "/" + slug + "/chart/top/artists", "/" + slug + "/chart/top/albums", "/" + slug + "/chart/top/tracks", "/" + slug + `/artist/:id(\d+)`, "/" + slug + `/album/:id(\d+)`, "/" + slug + `/track/:id(\d+)`}
		if len(clauses) != 2 || len(clauses[0].Href) != len(expected) || !strings.Contains(clauses[1].Not.Selector, "[download]") {
			t.Fatal(text)
		}
		for i, p := range clauses[0].Href {
			if p.Pathname != expected[i] || p.Search != "" {
				t.Fatalf("unexpected allowed route: %+v", p)
			}
			for _, forbidden := range []string{"logout", "delete", "manage", "admin", "apis", "?", "*", "://"} {
				if strings.Contains(p.Pathname, forbidden) {
					t.Fatalf("forbidden route %s", p.Pathname)
				}
			}
		}
	}
}

func TestInterest075DetailsAndTable(t *testing.T) {
	// Every interval exposes its exact values both natively in SVG and in the accessible table.
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	graph, err := buildInterest([]db.InterestBucket{{BucketStart: from, BucketEnd: from.Add(time.Hour), ListenCount: 3}, {BucketStart: from.Add(time.Hour), BucketEnd: from.Add(2 * time.Hour), ListenCount: 2}}, time.UTC, "detail")
	if err != nil {
		t.Fatal(err)
	}
	tpl, err := loadTemplates("/web-assets")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err = tpl.ExecuteTemplate(&out, "partial/interest-graph", graph); err != nil {
		t.Fatal(err)
	}
	body := out.String()
	for _, marker := range []string{`<details class="graph-values">`, `<summary>Interest values</summary>`, `<table>`, `scope="col"`, `<circle cx="8.000"`, `<title>`, `data-tip=`, `3 plays`, `2 plays`, `(inclusive)`} {
		if !strings.Contains(body, marker) {
			t.Fatalf("missing %s", marker)
		}
	}
	if strings.Count(body, "<circle") != 2 || strings.Count(body, "data-tip=") != 2 {
		t.Fatal("interval details lost")
	}
}

func TestSpeculation075SessionRace(t *testing.T) {
	// A ruleset created before login must never fetch personalized HTML after login.
	h, err := NewHandler(pageStore(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, purpose := range []string{"prefetch", "prefetch;prerender"} {
		r := httptest.NewRequest("GET", "/alpha/track/1", nil)
		r.Header.Set("Cookie", "tunetrail_session=any-account")
		r.Header.Set("Sec-Purpose", purpose)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 204 || w.Body.Len() != 0 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("personalized speculation: %d %s", w.Code, w.Body.String())
		}
	}
}
