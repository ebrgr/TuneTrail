package webui

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"html"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/go-chi/chi/v5"
)

func TestActivityCalendarTimezoneWeekStartAndThresholds(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 4, 1, 0, 0, 0, time.UTC) // Tuesday evening in Sao Paulo.
	items := []db.ListenActivityItem{{Start: now, Listens: 2}, {Start: now.Add(-time.Hour), Listens: 8}}
	for _, tc := range []struct {
		target  int64
		opacity string
	}{{100, "0.190"}, {20, "0.550"}, {10, "1.000"}, {5, "1.000"}} {
		grid, err := buildActivity(items, 4, loc, 1, tc.target, now)
		if err != nil || len(grid.Cells) != 252 || grid.Streak != 4 || grid.DayLabels[0] != "Mon" {
			t.Fatalf("grid: %v", err)
		}
		cell := grid.Cells[246]
		if cell.Date.Format("2006-01-02") != "2026-11-03" || cell.Count != 10 || cell.Opacity != tc.opacity {
			t.Fatalf("timezone/intensity: %+v", cell)
		}
		futures := 0
		for _, cell := range grid.Cells {
			if cell.Future {
				futures++
				if cell.Count != 0 {
					t.Fatal("future cell has activity")
				}
			}
		}
		if futures != 5 {
			t.Fatalf("future days: %d", futures)
		}
		if grid.Cells[69].Week != 9 || grid.Cells[70].Week != 10 {
			t.Fatal("mobile ten-week cut lost")
		}
	}
	for start := 0; start < 7; start++ {
		grid, err := buildActivity(nil, 0, loc, start, 100, now)
		if err != nil || int(grid.Cells[0].Date.Weekday()) != start {
			t.Fatal("week-start alignment")
		}
	}
}

func TestActivityUsesCalendarDaysAcrossDSTAndRejectsInvalidCounts(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	grid, err := buildActivity(nil, 0, loc, 1, 5, now)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	short, long := false, false
	for i, c := range grid.Cells {
		key := c.Date.Format("2006-01-02")
		if seen[key] || c.Date.Hour() != 0 {
			t.Fatal("duplicate or shifted calendar day")
		}
		seen[key] = true
		if i > 0 {
			hours := c.Date.Sub(grid.Cells[i-1].Date).Hours()
			short = short || hours == 23
			long = long || hours == 25
		}
	}
	if !short || !long {
		t.Fatal("DST boundaries were not represented")
	}
	for _, items := range [][]db.ListenActivityItem{{{Start: now, Listens: -1}}, {{Start: now, Listens: math.MaxInt64}, {Start: now, Listens: 1}}} {
		if _, err := buildActivity(items, 0, loc, 1, 5, now); err == nil {
			t.Fatal("invalid count became a valid chart")
		}
	}
}

type activityProbe struct {
	activityOpts, streakOpts db.ListenActivityOpts
	failStreak               bool
}

func (p *activityProbe) GetListenActivity(_ context.Context, o db.ListenActivityOpts) ([]db.ListenActivityItem, error) {
	p.activityOpts = o
	return nil, nil
}
func (p *activityProbe) GetListenStreak(_ context.Context, o db.ListenActivityOpts) (int, error) {
	p.streakOpts = o
	if p.failStreak {
		return 0, errors.New("private streak error")
	}
	return 3, nil
}

// TestActivityReaderPreservesFilterAndLocalTimezone keeps process timezone semantics.
func TestActivityReaderPreservesFilterAndLocalTimezone(t *testing.T) {
	probe := &activityProbe{}
	grid, err := readActivity(context.Background(), probe, db.ListenActivityOpts{ArtistID: 7, Timezone: time.FixedZone("Local", 0), Year: 1999, Month: 12, Range: 1, Step: db.StepYear}, 0, 20, time.Now())
	if err != nil || grid.Streak != 3 || grid.Timezone != "Local" {
		t.Fatal("activity read contract")
	}
	for _, o := range []db.ListenActivityOpts{probe.activityOpts, probe.streakOpts} {
		if o.ArtistID != 7 || o.AlbumID != 0 || o.TrackID != 0 || o.Timezone.String() != "Local" || o.Step != db.StepDay || o.Range != 252 || o.Year != 0 || o.Month != 0 {
			t.Fatalf("filter/date options: %+v", o)
		}
	}
	probe.failStreak = true
	if grid, err := readActivity(context.Background(), probe, db.ListenActivityOpts{}, 0, 100, time.Now()); err == nil || grid != nil {
		t.Fatal("failed streak became zero")
	}
}

func TestInterestSVGFiniteBoundedCurveAndAccessibleIntervals(t *testing.T) {
	for _, counts := range [][]int64{{}, {0, 0}, {3}, {5, 5, 5, 5}, {0, 100, 0, 100, 0}, {math.MaxInt64, 0}} {
		items := []db.InterestBucket{}
		for i, c := range counts {
			items = append(items, db.InterestBucket{BucketStart: time.Unix(int64(i), 0), BucketEnd: time.Unix(int64(i+1), 0), ListenCount: c})
		}
		graph, err := buildInterest(items, time.UTC, "test-interest")
		if err != nil {
			t.Fatal(err)
		}
		if len(graph.Rows) != len(items) {
			t.Fatal("table lost intervals")
		}
		for _, invalid := range []string{"NaN", "Inf"} {
			if strings.Contains(graph.Line, invalid) || strings.Contains(graph.Area, invalid) {
				t.Fatal("nonfinite SVG")
			}
		}
		if graph.Total == 0 && (!graph.Empty || graph.Line != "") {
			t.Fatal("empty chart fabricated a path")
		}
		if len(counts) > 1 && graph.Total > 0 && !strings.Contains(graph.Line, " C") {
			t.Fatal("bounded curve missing")
		}
		if len(counts) == 1 && graph.Total > 0 && graph.SingleX == "" {
			t.Fatal("single value is invisible")
		}
	}
	bad := []db.InterestBucket{{BucketStart: time.Unix(1, 0), BucketEnd: time.Unix(0, 0), ListenCount: 1}}
	if _, err := buildInterest(bad, time.UTC, "test"); err == nil {
		t.Fatal("reversed interval accepted")
	}
}

type graphProbe struct {
	ReadStore
	activityOpts []db.ListenActivityOpts
	interestOpts []db.GetInterestOpts
}

func (p *graphProbe) GetListenActivity(ctx context.Context, o db.ListenActivityOpts) ([]db.ListenActivityItem, error) {
	p.activityOpts = append(p.activityOpts, o)
	return p.ReadStore.GetListenActivity(ctx, o)
}
func (p *graphProbe) GetInterest(ctx context.Context, o db.GetInterestOpts) ([]db.InterestBucket, error) {
	p.interestOpts = append(p.interestOpts, o)
	return p.ReadStore.GetInterest(ctx, o)
}

func TestHomeAndMediaGraphsUseOnlyTheirOwnFiltersAndCounts(t *testing.T) {
	s := pageStore(t)
	probe := &graphProbe{ReadStore: s}
	h, err := NewHandler(probe, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"home", "artist", "album", "track"} {
		path := "/alpha?tz=UTC"
		if kind != "home" {
			path = "/alpha/" + kind + "/1?tz=UTC"
		}
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Cookie", "tunetrail-weekstart=1")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		o := probe.activityOpts[len(probe.activityOpts)-1]
		if o.ArtistID != map[string]int32{"artist": 1}[kind] || o.AlbumID != map[string]int32{"album": 1}[kind] || o.TrackID != map[string]int32{"track": 1}[kind] || o.Timezone != time.UTC {
			t.Fatalf("activity escaped its entity: %+v", o)
		}
		if !strings.Contains(w.Body.String(), "Activity values") || !strings.Contains(w.Body.String(), "Daily listening activity · UTC") {
			t.Fatal("missing non-JS activity values")
		}
		if kind == "home" {
			if len(probe.interestOpts) != 0 {
				t.Fatal("home gained unrelated interest query")
			}
			continue
		}
		interest := probe.interestOpts[len(probe.interestOpts)-1]
		if interest.Buckets != 16 || interest.ArtistID != o.ArtistID || interest.AlbumID != o.AlbumID || interest.TrackID != o.TrackID {
			t.Fatal("interest escaped its entity")
		}
		text := html.UnescapeString(w.Body.String())
		if strings.Contains(text, "ZgotmplZ") || !strings.Contains(text, "Interest values") || !strings.Contains(text, "(inclusive)") {
			t.Fatal("inaccessible or invalid graph markup")
		}
		start := strings.Index(text, `<svg viewBox="0 0 450 120"`)
		if start < 0 {
			t.Fatal("missing SVG")
		}
		end := strings.Index(text[start:], "</svg>") + start + len("</svg>")
		var svg any
		if err := xml.Unmarshal([]byte(text[start:end]), &svg); err != nil {
			t.Fatalf("invalid SVG XML: %v", err)
		}
	}
	for _, profile := range []struct {
		slug  string
		id    int
		total int64
	}{{"alpha", 1, 2}, {"beta", 2, 1}} {
		p := LivePage{Page: Page{Profile: &Profile{ID: profile.id, Slug: profile.slug}}, Options: db.GetItemsOpts{Timeframe: db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC}}, Errors: map[string]bool{}}
		if err := h.loadMedia(db.WithProfile(context.Background(), int32(profile.id)), &p, "track", 1); err != nil {
			t.Fatal(err)
		}
		if p.Media.Interest.Total != profile.total {
			t.Fatal("interest leaked across profiles")
		}
		var activityTotal int64
		for _, c := range p.Activity.Cells {
			activityTotal += c.Count
		}
		if activityTotal != profile.total {
			t.Fatal("activity leaked across profiles")
		}
	}
}

type brokenGraphs struct {
	ReadStore
	streakOnly bool
}

func (s brokenGraphs) GetListenActivity(ctx context.Context, o db.ListenActivityOpts) ([]db.ListenActivityItem, error) {
	if s.streakOnly {
		return s.ReadStore.GetListenActivity(ctx, o)
	}
	return nil, errors.New("private activity SQL")
}
func (s brokenGraphs) GetListenStreak(context.Context, db.ListenActivityOpts) (int, error) {
	return 0, errors.New("private streak SQL")
}
func (s brokenGraphs) GetInterest(context.Context, db.GetInterestOpts) ([]db.InterestBucket, error) {
	return nil, errors.New("private interest SQL")
}
func TestGraphFailuresPreserveHeadersWithoutFakeZeros(t *testing.T) {
	s := pageStore(t)
	for _, streakOnly := range []bool{false, true} {
		h, err := NewHandler(brokenGraphs{s, streakOnly}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{"/alpha", "/alpha/artist/1"} {
			w := pageRequest(h, path)
			body := w.Body.String()
			if w.Code != 200 || !strings.Contains(body, `data-listen-count="3"`) || !strings.Contains(body, "Activity unavailable") || strings.Contains(body, "private ") || strings.Contains(body, "Activity values") {
				t.Fatal("failed graph became valid data")
			}
			if strings.Contains(path, "artist") && (!strings.Contains(body, "Interest unavailable") || strings.Contains(body, "Interest values")) {
				t.Fatal("interest failure became zeros")
			}
		}
	}
}

func interestAPI(store db.ListenStore, profile int32, kind, id, query string) *httptest.ResponseRecorder {
	route := chi.NewRouteContext()
	route.URLParams.Add("id", id)
	ctx := context.WithValue(db.WithProfile(context.Background(), profile), chi.RouteCtxKey, route)
	req := httptest.NewRequest("GET", "/?"+query, nil).WithContext(ctx)
	w := httptest.NewRecorder()
	switch kind {
	case "artist":
		handlers.GetArtistInterestHandler(store)(w, req)
	case "album":
		handlers.GetAlbumInterestHandler(store)(w, req)
	case "track":
		handlers.GetTrackInterestHandler(store)(w, req)
	}
	return w
}
func TestInterestAPIValidationAndHTMLParity(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		for _, query := range []string{"", "buckets=0", "buckets=-1", "buckets=513", "buckets=999999999999999999999", "buckets=abc"} {
			if got := interestAPI(nil, 1, kind, "1", query).Code; got != http.StatusBadRequest {
				t.Fatalf("%s %s: %d", kind, query, got)
			}
		}
		for _, id := range []string{"0", "-1", "abc", "2147483648"} {
			if interestAPI(nil, 1, kind, id, "buckets=16").Code != 400 {
				t.Fatal("invalid ID reached store")
			}
		}
	}
	s := pageStore(t)
	for _, kind := range []string{"artist", "album", "track"} {
		for _, profile := range []struct {
			slug string
			id   int
		}{{"alpha", 1}, {"beta", 2}} {
			response := interestAPI(s, int32(profile.id), kind, "1", "buckets=16")
			if response.Code != 200 {
				t.Fatal(response.Body.String())
			}
			var items []db.InterestBucket
			if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
				t.Fatal(err)
			}
			h, err := NewHandler(s, Options{})
			if err != nil {
				t.Fatal(err)
			}
			body := pageRequest(h, "/"+profile.slug+"/"+kind+"/1").Body.String()
			var sum int64
			for _, item := range items {
				sum += item.ListenCount
			}
			unit := " plays in 16 intervals"
			if sum == 1 {
				unit = " play in 16 intervals"
			}
			if !strings.Contains(body, strconv.FormatInt(sum, 10)+unit) {
				t.Fatal("HTML/API interest counts differ")
			}
		}
	}
	bad := interestAPI(brokenGraphs{ReadStore: s}, 1, "track", "1", "buckets=16")
	if bad.Code != 500 || strings.Contains(bad.Body.String(), "private ") {
		t.Fatal("interest API error leaked or hidden")
	}
}

// TestActivityGridUsesCivilDatesWhenMidnightIsSkipped avoids a duplicated previous day.
func TestActivityGridUsesCivilDatesWhenMidnightIsSkipped(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2018, 11, 4, 1, 30, 0, 0, loc)
	grid, err := buildActivity([]db.ListenActivityItem{{Start: at, Listens: 3}}, 0, loc, 1, 10, time.Date(2018, 11, 5, 12, 0, 0, 0, loc))
	if err != nil {
		t.Fatal(err)
	}
	matches := 0
	for _, cell := range grid.Cells {
		if cell.Date.Format("2006-01-02") == "2018-11-04" {
			matches++
			if cell.Count != 3 || cell.Date.Hour() != 1 {
				t.Fatalf("bad DST cell: %+v", cell)
			}
		}
	}
	if matches != 1 {
		t.Fatalf("expected one civil date, got %d", matches)
	}
}
