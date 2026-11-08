package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func pageStore(t *testing.T) *sqlite.Sqlite {
	t.Helper()
	if err := cfg.Load(func(string) string { return "" }, "test"); err != nil {
		t.Fatal(err)
	}
	s, err := sqlite.NewInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(context.Background()) })
	for _, q := range []string{
		`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','A <script>alert(1)</script>',1),(2,'beta','Beta',1),(3,'empty','Empty',1)`,
		`INSERT INTO artists(id) VALUES(1),(2)`,
		`INSERT INTO artist_aliases VALUES(1,'Shared Artist','User',1),(2,'Second Artist','User',1)`,
		`INSERT INTO releases(id,various_artists) VALUES(1,1),(2,0)`,
		`INSERT INTO release_aliases VALUES(1,'Shared Album','User',1),(2,'Only Beta Album','User',1)`,
		`INSERT INTO artist_releases VALUES(1,1,1),(2,2,1)`,
		`INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180),(2,1,240),(3,2,120)`,
		`INSERT INTO track_aliases VALUES(1,'Alpha Track','User',1),(2,'Other Alpha Track','User',1),(3,'Exclusive Beta Track','User',1)`,
		`INSERT INTO artist_tracks VALUES(1,1,1),(1,2,1),(2,3,1)`,
	} {
		if err := s.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().Add(-time.Hour).Truncate(time.Second)
	for _, v := range []struct {
		p, tr int32
		shift int
	}{{1, 1, 0}, {1, 2, 0}, {1, 1, 1}, {2, 1, 0}, {2, 3, 2}} {
		if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: v.p, TrackID: v.tr, Time: now.Add(time.Duration(v.shift) * time.Minute)}); err != nil {
			t.Fatal(err)
		}
	}
	return s
}
func pageRequest(h http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}
func TestLivePagesUseScopedSQLiteAndAPIValues(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/", "/alpha?period=all_time&tz=UTC", "/alpha/chart/top/artists?period=all_time", "/alpha/chart/top/albums", "/alpha/chart/top/tracks", "/alpha/listens?limit=1&page=2", "/empty", "/empty/chart/top/tracks"} {
		w := pageRequest(h, path)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		body := w.Body.String()
		if strings.Contains(body, "ZgotmplZ") || strings.Contains(body, "Exclusive Beta Track") || strings.Contains(body, "<script>alert(1)</script>") {
			t.Fatalf("scope/escaping: %s", path)
		}
	}
	body := pageRequest(h, "/alpha?period=all_time&tz=UTC").Body.String()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/?period=all_time&tz=UTC", nil)
	req = req.WithContext(db.WithProfile(req.Context(), 1))
	handlers.StatsHandler(s).ServeHTTP(w, req)
	var stats handlers.StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if stats.ListenCount != 3 || !strings.Contains(body, fmt.Sprintf(`data-listen-count="%d"`, stats.ListenCount)) {
		t.Fatal("HTML/API statistics mismatch")
	}
	if !strings.Contains(body, "Various Artists") || !strings.Contains(body, `href="/alpha/track/1"`) {
		t.Fatal("catalog links/compilation lost")
	}
	for _, path := range []string{"/missing", "/alpha/unknown"} {
		if w := pageRequest(h, path); w.Code != 404 {
			t.Fatalf("expected 404: %s", path)
		}
	}
	if w := pageRequest(h, "/alpha/listens?limit=1&page=20"); !strings.Contains(w.Body.String(), "Page 20") || !strings.Contains(w.Body.String(), "Nothing to show on this page") {
		t.Fatal("out-of-range page misrepresented")
	}
}

type brokenReads struct{ ReadStore }

func (s brokenReads) CountListens(context.Context, db.Timeframe) (int64, error) {
	return 0, errors.New("private SQL detail")
}

type brokenProfiles struct{ ReadStore }

func (s brokenProfiles) GetListeningProfile(context.Context, string) (*models.ListeningProfile, error) {
	return nil, errors.New("database unavailable")
}
func TestLiveErrorsDoNotBecomeFakeZerosOr404(t *testing.T) {
	s := pageStore(t)
	h, e := NewHandler(brokenReads{s}, Options{})
	if e != nil {
		t.Fatal(e)
	}
	w := pageRequest(h, "/alpha")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Statistics unavailable") || strings.Contains(w.Body.String(), "private SQL detail") {
		t.Fatal("block error not handled")
	}
	h, e = NewHandler(brokenProfiles{s}, Options{})
	if e != nil {
		t.Fatal(e)
	}
	if pageRequest(h, "/alpha").Code != 500 {
		t.Fatal("infrastructure error must not become 404")
	}
}

func TestLivePagingFiltersAndSameSecond(t *testing.T) {
	s := pageStore(t)
	h, e := NewHandler(s, Options{})
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/alpha/listens?limit=-1", "/alpha/listens?month=13", "/alpha/chart/top/artists?artist_id=1", "/alpha?year=2026"} {
		if pageRequest(h, path).Code != 400 {
			t.Fatal("invalid filter accepted: " + path)
		}
	}
	for _, path := range []string{"/alpha/listens?page=0&limit=1", "/alpha/chart/top/tracks?album_id=1&limit=1&page=2&period=all_time&tz=UTC"} {
		w := pageRequest(h, path)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `href="/alpha/`) {
			t.Fatal("missing scope")
		}
	}
	pages := []int32{}
	for pg := 1; pg <= 3; pg++ {
		r, e := s.GetListensPaginated(db.WithProfile(context.Background(), 1), db.GetItemsOpts{Page: pg, Limit: 1, Timeframe: db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC}})
		if e != nil {
			t.Fatal(e)
		}
		pages = append(pages, r.Items[0].Track.ID)
	}
	if pages[1] != 2 || pages[2] != 1 {
		t.Fatalf("same-second order: %v", pages)
	}
}

func TestRoutingSearchAndLegacy(t *testing.T) {
	data, e := os.ReadFile("../../test_assets/web-ui-routes.json")
	if e != nil {
		t.Fatal(e)
	}
	var cases [][]any
	if e = json.Unmarshal(data, &cases); e != nil {
		t.Fatal(e)
	}
	for _, v := range cases {
		if IsPage(v[0].(string)) != v[1].(bool) {
			t.Fatal(v[0])
		}
	}
	s := pageStore(t)
	h, e := NewHandler(s, Options{})
	if e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"Exclusive+Beta+Track", "id:3"} {
		w := pageRequest(h, "/alpha/search?q="+q)
		if w.Code != 200 || strings.Contains(w.Body.String(), "Exclusive Beta Track</a>") {
			t.Fatal("search leaked profile")
		}
	}
	if w := pageRequest(h, "/beta/search?q=Exclusive"); w.Code != 200 || !strings.Contains(w.Body.String(), "Exclusive Beta Track") {
		t.Fatal("search failed")
	}
	for _, next := range []string{"https://example.org", "//example.org", "javascript:alert(1)", "/admin", "/../admin", "/artist/nope", "listens", "artist/1"} {
		if safeNext(next) != "" {
			t.Fatal("unsafe next")
		}
	}
	if safeNext("/chart/top/tracks?period=all_time") == "" {
		t.Fatal("valid next rejected")
	}
	w := httptest.NewRecorder()
	if !h.RedirectLegacy(w, httptest.NewRequest("GET", "/listens?period=year", nil)) || !strings.HasPrefix(w.Header().Get("Location"), "/?next=") {
		t.Fatal("legacy selection")
	}
	if got := switchURL("beta", "/alpha/chart/top/tracks", url.Values{"period": {"month"}, "page": {"5"}, "track_id": {"1"}}); got != "/beta/chart/top/tracks?period=month" {
		t.Fatal(got)
	}
}

func TestNowPlayingIsScopedAndMissingTrackClearsOnlyItsProfile(t *testing.T) {
	s := pageStore(t)
	memkv.Store.Set("now-playing:1", int32(1))
	memkv.Store.Set("now-playing:2", int32(3))
	t.Cleanup(func() { memkv.Store.Delete("now-playing:1"); memkv.Store.Delete("now-playing:2") })
	a, e := listening.ReadNowPlaying(db.WithProfile(context.Background(), 1), s)
	if e != nil || !a.CurrentlyPlaying || a.Track.ID != 1 {
		t.Fatalf("alpha playing: %+v %v", a, e)
	}
	b, e := listening.ReadNowPlaying(db.WithProfile(context.Background(), 2), s)
	if e != nil || !b.CurrentlyPlaying || b.Track.ID != 3 {
		t.Fatalf("beta playing: %+v %v", b, e)
	}
	memkv.Store.Set("now-playing:1", int32(99999))
	a, e = listening.ReadNowPlaying(db.WithProfile(context.Background(), 1), s)
	if e != nil || a.CurrentlyPlaying {
		t.Fatalf("missing track: %+v %v", a, e)
	}
	if _, ok := memkv.Store.Get("now-playing:1"); ok {
		t.Fatal("stale key retained")
	}
	if _, ok := memkv.Store.Get("now-playing:2"); !ok {
		t.Fatal("other profile cache cleared")
	}
}
