package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/go-chi/chi/v5"
)

func mediaAPI(t *testing.T, s ReadStore, profile int32, kind, id string) *httptest.ResponseRecorder {
	t.Helper()
	route := chi.NewRouteContext()
	route.URLParams.Add("id", id)
	ctx := context.WithValue(db.WithProfile(context.Background(), profile), chi.RouteCtxKey, route)
	req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	var handler http.HandlerFunc
	switch kind {
	case "artist":
		handler = handlers.GetArtistHandler(s)
	case "album":
		handler = handlers.GetAlbumHandler(s)
	case "track":
		handler = handlers.GetTrackHandler(s)
	}
	handler(w, req)
	return w
}

func TestMediaHeadersMatchAPIAndKeepProfileScope(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []struct {
		slug string
		id   int32
	}{{"alpha", 1}, {"beta", 2}} {
		for _, kind := range []string{"artist", "album", "track"} {
			path := "/" + profile.slug + "/" + kind + "/1?tz=UTC"
			w := pageRequest(h, path)
			if w.Code != 200 {
				t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
			}
			api := mediaAPI(t, s, profile.id, kind, "1")
			if api.Code != 200 {
				t.Fatal(api.Body.String())
			}
			var data map[string]any
			if err := json.Unmarshal(api.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			for key, field := range map[string]string{"data-listen-count": "listen_count", "data-time-listened": "time_listened", "data-rank": "all_time_rank"} {
				want := fmt.Sprintf(`%s="%.0f"`, key, data[field].(float64))
				if !strings.Contains(w.Body.String(), want) {
					t.Fatalf("%s missing %s", path, want)
				}
			}
			first := time.Unix(int64(data["first_listen"].(float64)), 0).UTC().Format(time.RFC3339)
			if !strings.Contains(w.Body.String(), `datetime="`+first+`"`) {
				t.Fatalf("first listen mismatch %s", path)
			}
			if profile.slug == "alpha" && strings.Contains(w.Body.String(), "Exclusive Beta Track") {
				t.Fatal("cross-profile content")
			}
			if strings.Contains(w.Body.String(), "<script>alert(1)</script>") || strings.Contains(w.Body.String(), "ZgotmplZ") {
				t.Fatal("unsafe rendering")
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("public media must not be cached across profiles")
			}
		}
	}
	for _, path := range []string{"/alpha/artist/2", "/alpha/album/2", "/alpha/track/3", "/empty/track/1", "/missing/artist/1", "/alpha/track/999"} {
		if got := pageRequest(h, path).Code; got != 404 {
			t.Fatalf("%s: %d", path, got)
		}
	}
	// Public media routes are dispatched to the scoped Go handler.
	for _, path := range []string{"/alpha/artist/1", "/alpha/album/1", "/alpha/track/1"} {
		if !IsPage(path) {
			t.Fatal("media route not activated")
		}
	}
}

func TestMediaFiltersAndIDsCannotReplaceRouteIdentity(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"artist", "album", "track"} {
		for _, id := range []string{"0", "-1", "nope", "2147483648"} {
			path := "/alpha/" + kind + "/" + id
			if got := pageRequest(h, path).Code; got != 400 {
				t.Fatalf("%s: %d", path, got)
			}
			if got := mediaAPI(t, s, 1, kind, id).Code; got != 400 {
				t.Fatalf("API %s: %d", id, got)
			}
		}
	}
	for _, query := range []string{"period=month", "year=2026", "page=2", "limit=500", "artist_id=2", "track_id=3", "from=1", "tz=UTC&tz=Asia/Tokyo"} {
		if pageRequest(h, "/alpha/track/1?"+query).Code != 400 {
			t.Fatal("unsupported filter accepted: " + query)
		}
	}
	for _, path := range []string{"/alpha/track/1/extra", "/alpha/artist"} {
		if pageRequest(h, path).Code != 404 {
			t.Fatal("invalid shape")
		}
	}
}

func TestMediaRelationsImagesAndExternalLinks(t *testing.T) {
	s := pageStore(t)
	mbid := "00000000-0000-4000-8000-000000000123"
	for _, sql := range []string{
		`UPDATE artists SET musicbrainz_id='` + mbid + `' WHERE id=1`,
		`UPDATE releases SET musicbrainz_id='` + mbid + `' WHERE id=1`,
		`UPDATE tracks SET musicbrainz_id='` + mbid + `' WHERE id=1`,
		`UPDATE track_aliases SET alias='<img src=x onerror=alert(1)>' WHERE track_id=1`,
		`INSERT INTO artist_tracks VALUES(2,1,0)`,
	} {
		if err := s.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for kind, entity := range map[string]string{"artist": "artist", "album": "release", "track": "recording"} {
		body := pageRequest(h, "/alpha/"+kind+"/1").Body.String()
		if !strings.Contains(body, "https://musicbrainz.org/"+entity+"/"+mbid) {
			t.Fatal("wrong MBID entity: " + kind)
		}
		if strings.Contains(body, "<img src=x onerror=") {
			t.Fatal("catalog markup was not escaped")
		}
		if !strings.Contains(body, `src="/image/`) && !strings.Contains(body, `data-cover`) {
			t.Fatal("missing cover fallback")
		}
	}
	album := pageRequest(h, "/alpha/album/1").Body.String()
	if !strings.Contains(album, "By ") || !strings.Contains(album, "Various Artists") {
		t.Fatal("compilation metadata lost")
	}
	track := pageRequest(h, "/alpha/track/1").Body.String()
	for _, want := range []string{`href="/alpha/artist/1"`, `href="/alpha/artist/2"`, `href="/alpha/album/1"`, `href="/alpha/listens?period=all_time&amp;track_id=1"`} {
		if !strings.Contains(track, want) {
			t.Fatal("track relation/filter missing: " + want)
		}
	}
	withTimezone := html.UnescapeString(pageRequest(h, "/alpha/artist/1?tz=Asia/Tokyo").Body.String())
	if !strings.Contains(withTimezone, `/alpha/listens?artist_id=1&period=all_time&tz=Asia%2FTokyo`) || !strings.Contains(withTimezone, "+09:00") {
		t.Fatal("explicit timezone lost in date or history link")
	}
}

type mediaBrokenStore struct{ ReadStore }

func (s mediaBrokenStore) GetArtist(context.Context, db.GetArtistOpts) (*models.Artist, error) {
	return nil, errors.New("private database failure")
}
func (s mediaBrokenStore) GetAlbum(context.Context, db.GetAlbumOpts) (*models.Album, error) {
	return nil, errors.New("private database failure")
}
func (s mediaBrokenStore) GetTrack(context.Context, db.GetTrackOpts) (*models.Track, error) {
	return nil, errors.New("private database failure")
}

type mediaBrokenBlocks struct{ ReadStore }

func (s mediaBrokenBlocks) GetListensPaginated(context.Context, db.GetItemsOpts) (*db.PaginatedResponse[*models.Listen], error) {
	return nil, errors.New("private listens SQL")
}
func (s mediaBrokenBlocks) GetTopTracksPaginated(context.Context, db.GetItemsOpts) (*db.PaginatedResponse[db.RankedItem[*models.Track]], error) {
	return nil, errors.New("private tracks SQL")
}
func (s mediaBrokenBlocks) GetTopAlbumsPaginated(context.Context, db.GetItemsOpts) (*db.PaginatedResponse[db.RankedItem[*models.Album]], error) {
	return nil, errors.New("private albums SQL")
}

type missingTrackAlbum struct{ ReadStore }

func (s missingTrackAlbum) GetAlbum(context.Context, db.GetAlbumOpts) (*models.Album, error) {
	return nil, db.ErrNotFound
}

func TestMediaEssentialAndOptionalFailures(t *testing.T) {
	s := pageStore(t)
	broken := mediaBrokenStore{s}
	h, err := NewHandler(broken, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"artist", "album", "track"} {
		w := pageRequest(h, "/alpha/"+kind+"/1")
		if w.Code != 500 || strings.Contains(w.Body.String(), "private database") {
			t.Fatal("HTML database error hidden or leaked")
		}
		api := mediaAPI(t, broken, 1, kind, "1")
		if api.Code != 500 || strings.Contains(api.Body.String(), "private database") {
			t.Fatal("API database error hidden or leaked")
		}
		if mediaAPI(t, s, 1, kind, "999").Code != 404 {
			t.Fatal("API missing media must remain 404")
		}
	}
	h, err = NewHandler(mediaBrokenBlocks{s}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	w := pageRequest(h, "/alpha/artist/1")
	for _, want := range []string{`data-listen-count="3"`, "Ranking unavailable", "Listening history unavailable", "Albums unavailable"} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatal("missing surviving block/error: " + want)
		}
	}
	if w.Code != 200 || strings.Contains(w.Body.String(), "private ") {
		t.Fatal("optional failure corrupted public response")
	}
	h, err = NewHandler(missingTrackAlbum{s}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if pageRequest(h, "/alpha/track/1").Code != 500 {
		t.Fatal("broken track relation must not become a missing track")
	}
}

func TestMediaNowPlayingMatchesEntityAndProfile(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	memkv.Store.Set("now-playing:1", int32(1))
	memkv.Store.Set("now-playing:2", int32(3))
	t.Cleanup(func() { memkv.Store.Delete("now-playing:1"); memkv.Store.Delete("now-playing:2") })
	for _, tc := range []struct {
		path    string
		playing bool
	}{{"/alpha/artist/1", true}, {"/alpha/album/1", true}, {"/alpha/track/1", true}, {"/alpha/track/2", false}, {"/beta/track/1", false}, {"/beta/track/3", true}} {
		body := pageRequest(h, tc.path).Body.String()
		if strings.Contains(body, ">Now Playing") != tc.playing {
			t.Fatal("incorrect playing match: " + tc.path)
		}
	}
}

func TestMediaLimitsAndCompleteAlbumLink(t *testing.T) {
	s := pageStore(t)
	for id := 100; id < 200; id++ {
		for _, q := range []string{
			fmt.Sprintf(`INSERT INTO releases(id) VALUES(%d)`, id),
			fmt.Sprintf(`INSERT INTO release_aliases VALUES(%d,'Album %d','User',1)`, id, id),
			fmt.Sprintf(`INSERT INTO artist_releases VALUES(1,%d,1)`, id),
			fmt.Sprintf(`INSERT INTO tracks(id,release_id,duration) VALUES(%d,%d,180)`, id, id),
			fmt.Sprintf(`INSERT INTO track_aliases VALUES(%d,'Track %d','User',1)`, id, id),
			fmt.Sprintf(`INSERT INTO artist_tracks VALUES(1,%d,1)`, id),
		} {
			if err := s.Exec(q); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: int32(id), Time: time.Now().Add(-time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	p := LivePage{Page: Page{Profile: &Profile{ID: 1, Slug: "alpha"}}, Options: db.GetItemsOpts{Timeframe: db.Timeframe{Period: db.PeriodAllTime, Timezone: time.UTC}}, Errors: map[string]bool{}}
	if err := h.loadMedia(db.WithProfile(context.Background(), 1), &p, "artist", 1); err != nil {
		t.Fatal(err)
	}
	if len(p.Media.TopTracks.Items) != 8 || len(p.Listens) != 11 || len(p.Media.Albums) != 99 || !p.Media.AlbumsMore || !p.HistoryMore {
		t.Fatal("media limits or continuation lost")
	}
	body := pageRequest(h, "/alpha/artist/1").Body.String()
	for _, want := range []string{`/alpha/chart/top/albums?artist_id=1&amp;period=all_time`, `/alpha/chart/top/tracks?artist_id=1&amp;period=all_time`, `/alpha/listens?artist_id=1&amp;period=all_time`, "SEE ALL ALBUMS"} {
		if !strings.Contains(body, want) {
			t.Fatal("missing complete destination: " + want)
		}
	}
}

type noFirstListen struct{ ReadStore }

func (s noFirstListen) GetArtist(ctx context.Context, o db.GetArtistOpts) (*models.Artist, error) {
	a, e := s.ReadStore.GetArtist(ctx, o)
	if e == nil {
		a.FirstListen = 0
	}
	return a, e
}
func TestMediaNoFirstListenAndTiedRanks(t *testing.T) {
	s := pageStore(t)
	for _, q := range []string{`INSERT INTO artist_tracks VALUES(2,1,0)`, `INSERT INTO artist_tracks VALUES(2,2,0)`} {
		if err := s.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(noFirstListen{s}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"1", "2"} {
		body := pageRequest(h, "/alpha/artist/"+id).Body.String()
		if !strings.Contains(body, `data-rank="1"`) || strings.Contains(body, "Listening since") || strings.Contains(body, "1970") {
			t.Fatal("rank/absent date")
		}
	}
}
