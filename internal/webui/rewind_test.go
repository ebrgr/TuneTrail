package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/summary"
)

func TestRewindSQLiteAndSummaryAPI(t *testing.T) {
	s := pageStore(t)
	for _, listen := range []struct {
		profile, track int32
		at             string
	}{
		{1, 1, "2025-12-31T23:59:59Z"}, {1, 1, "2026-01-01T00:00:00Z"}, {1, 1, "2026-01-15T12:00:00Z"}, {1, 1, "2026-01-31T23:59:59Z"}, {1, 1, "2026-02-01T00:00:00Z"},
		{1, 3, "2026-01-20T00:00:00Z"}, {1, 3, "2026-01-21T00:00:00Z"}, {1, 3, "2026-01-22T00:00:00Z"}, {2, 1, "2026-01-05T00:00:00Z"},
	} {
		at, err := time.Parse(time.RFC3339, listen.at)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: listen.profile, TrackID: listen.track, Time: at}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(s, Options{Now: func() time.Time { return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	path := "/alpha/rewind?year=2026&month=1&tz=UTC"
	w := pageRequest(h, path)
	if w.Code != 200 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	req := httptest.NewRequest("GET", "/?year=2026&month=1&tz=UTC", nil)
	req = req.WithContext(db.WithProfile(req.Context(), 1))
	api := httptest.NewRecorder()
	handlers.SummaryHandler(s).ServeHTTP(api, req)
	var data summary.Summary
	if err := json.Unmarshal(api.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if api.Code != 200 || data.Plays != 6 || data.MinutesListened != 15 || data.UniqueTracks != 2 || data.UniqueAlbums != 2 || data.UniqueArtists != 2 || data.NewTracks != 1 || data.NewAlbums != 1 || data.NewArtists != 1 || math.Abs(float64(data.AvgPlaysPerDay)-6.0/31) > 0.00001 {
		t.Fatalf("unexpected API summary %d: %+v", api.Code, data)
	}
	if len(data.TopTracks) != 2 || len(data.TopAlbums) != 2 || len(data.TopArtists) != 2 {
		t.Fatalf("rankings: %+v", data)
	}
	for _, expected := range []string{`data-plays="6"`, `data-minutes="15"`, fmt.Sprintf(`data-average-plays="%.6f"`, data.AvgPlaysPerDay), "Your January 2026 Rewind", "Alpha Track", "Exclusive Beta Track", `name="tz" value="UTC"`, "&lt;script&gt;"} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %q", expected)
		}
	}
	if strings.Contains(body, "<script>alert(1)</script>") || strings.Contains(body, "ZgotmplZ") || strings.Contains(body, "(9 minutes)") {
		t.Fatal("invalid escaping or show_time off")
	}
	timed := pageRequest(h, path+"&show_time=1").Body.String()
	if !strings.Contains(timed, "(9 minutes)") || !strings.Contains(timed, `data-plays="6"`) {
		t.Fatal("show_time should change presentation only")
	}
	beta := pageRequest(h, "/beta/rewind?year=2026&month=1&tz=UTC").Body.String()
	if !strings.Contains(beta, `data-plays="1"`) || strings.Contains(beta, "Exclusive Beta Track") {
		t.Fatal("profile scope leaked")
	}
	empty := pageRequest(h, "/empty/rewind?year=2026&month=1&tz=UTC").Body.String()
	if !strings.Contains(empty, "Not enough data") {
		t.Fatal("empty period not shown")
	}
	for _, bad := range []string{"/alpha/rewind?year=2026&month=13", "/alpha/rewind?year=2027", "/alpha/rewind?show_time=2", "/alpha/rewind?year=2025&year=2026"} {
		if got := pageRequest(h, bad).Code; got != 400 {
			t.Errorf("%s: %d", bad, got)
		}
	}
	if !IsPage("/alpha/rewind") {
		t.Fatal("rewind must be dispatched to Go")
	}
}

type brokenRewind struct{ ReadStore }

func (s brokenRewind) GetFirstListenUnix(context.Context) (int64, error) {
	return 0, errors.New("private SQL detail")
}
func TestRewindFirstListenError(t *testing.T) {
	h, err := NewHandler(brokenRewind{pageStore(t)}, Options{Now: func() time.Time { return time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	w := pageRequest(h, "/alpha/rewind?year=2026&month=1&tz=UTC")
	if w.Code != 500 || strings.Contains(w.Body.String(), "private SQL detail") {
		t.Fatalf("unexpected error response: %d %s", w.Code, w.Body.String())
	}
}

func TestRewindLimitsRankingsToFive(t *testing.T) {
	s := pageStore(t)
	for i := 4; i <= 9; i++ {
		for _, q := range []string{
			fmt.Sprintf(`INSERT INTO artists(id) VALUES(%d)`, i),
			fmt.Sprintf(`INSERT INTO artist_aliases VALUES(%d,'Artist %d','User',1)`, i, i),
			fmt.Sprintf(`INSERT INTO releases(id,various_artists) VALUES(%d,0)`, i),
			fmt.Sprintf(`INSERT INTO release_aliases VALUES(%d,'Album %d','User',1)`, i, i),
			fmt.Sprintf(`INSERT INTO artist_releases VALUES(%d,%d,1)`, i, i),
			fmt.Sprintf(`INSERT INTO tracks(id,release_id,duration) VALUES(%d,%d,60)`, i, i),
			fmt.Sprintf(`INSERT INTO track_aliases VALUES(%d,'Track %d','User',1)`, i, i),
			fmt.Sprintf(`INSERT INTO artist_tracks VALUES(%d,%d,1)`, i, i),
		} {
			if err := s.Exec(q); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: int32(i), Time: time.Date(2026, 1, i, 0, 0, 0, 0, time.UTC)}); err != nil {
			t.Fatal(err)
		}
	}
	result, err := summary.GenerateSummary(context.Background(), s, 1, db.Timeframe{Year: 2026, Month: 1, Timezone: time.UTC}, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Plays != 6 || len(result.TopArtists) != 5 || len(result.TopAlbums) != 5 || len(result.TopTracks) != 5 {
		t.Fatalf("ranking limits: %+v", result)
	}
	for _, entry := range result.TopTracks {
		if entry.Item.ListenCount != 1 || entry.Item.TimeListened != 60 {
			t.Fatalf("tie stats: %+v", entry)
		}
	}
}

// TestRewindAdjacentYears preserves presentation and clamps unavailable months.
func TestRewindAdjacentYears(t *testing.T) {
	now := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	first := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	p := &RewindPage{Year: 2025, Month: 12}
	buildRewindChoices(p, url.Values{"tz": {"UTC"}, "show_time": {"1"}}, "/alpha/rewind", now, &first)
	if p.PreviousYear.URL != "" {
		t.Fatal("previous year before first listen")
	}
	u, err := url.Parse(p.NextYear.URL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("year") != "2026" || u.Query().Get("month") != "10" || u.Query().Get("tz") != "UTC" || u.Query().Get("show_time") != "1" {
		t.Fatalf("invalid next year URL: %s", u)
	}
	p = &RewindPage{Year: 2026, Month: 10}
	buildRewindChoices(p, url.Values{"tz": {"UTC"}}, "/alpha/rewind", now, &first)
	u, err = url.Parse(p.PreviousYear.URL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("year") != "2025" || u.Query().Get("month") != "10" || p.NextYear.URL != "" {
		t.Fatalf("invalid year boundary: %+v", p)
	}
}
