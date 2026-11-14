package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// referenceStore fixes the interest clock seam without changing production aggregation.
type referenceStore struct{ *sqlite.Sqlite }

func (s referenceStore) GetInterest(ctx context.Context, o db.GetInterestOpts) ([]db.InterestBucket, error) {
	buckets, err := s.Sqlite.GetInterest(ctx, o)
	if err != nil {
		return nil, err
	}
	var count int64
	for _, b := range buckets {
		count += b.ListenCount
	}
	return []db.InterestBucket{{BucketStart: time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), BucketEnd: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC), ListenCount: count}}, nil
}

// referenceSettings supplies a fictitious, stable server-settings snapshot.
type referenceSettings struct{}

func (referenceSettings) Read(context.Context) (cfg.SettingsSnapshot, error) {
	return cfg.SettingsSnapshot{Revision: strings.Repeat("a", 64), Saved: map[string]string{}, Effective: map[string]string{}, Active: map[string]string{}, Origins: map[string]cfg.Origin{}, SavedOrigins: map[string]cfg.Origin{}}, nil
}
func (referenceSettings) Save(context.Context, string, map[string]string) (cfg.SettingsSnapshot, error) {
	return referenceSettings{}.Read(context.Background())
}

// TestUIReference exports real handlers against an isolated, fixed-time fixture when explicitly requested.
func TestUIReference(t *testing.T) {
	out := os.Getenv("TUNETRAIL_UI_REFERENCE")
	if out == "" {
		t.Skip("set TUNETRAIL_UI_REFERENCE to an empty output directory")
	}
	// Make exports the developer .env; never let operational configuration enter this fixture.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if (strings.HasPrefix(key, "TUNETRAIL_") || strings.HasPrefix(key, "KOITO_")) && key != "TUNETRAIL_UI_REFERENCE" {
			t.Setenv(key, "")
		}
	}
	if err := os.Mkdir(out, 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		p := filepath.Join(out, name)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	s := pageStore(t)
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	if err := s.Exec(`DELETE FROM listens; INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,1714564800),(1,2,1714564800),(2,1,1714564800),(2,3,1714564800)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Exec(`UPDATE listening_profiles SET display_name='Fictional Alpha' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	_, err := s.SaveUser(context.Background(), db.SaveUserOpts{Username: "reference-admin", Password: "fictional-password", Role: models.UserRoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	live, err := NewHandler(referenceStore{s}, Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	// The production preview clock is intentionally not injectable. Its test fixture pins calendar cells.
	previewTemplates, err := loadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	preview := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := makePage(r)
		p.Cells = nil
		today := time.Date(2024, 6, 1, 0, 0, 0, 0, time.FixedZone("America/Sao_Paulo", -3*3600))
		start := today.AddDate(0, 0, -int(today.Weekday())-35*7)
		for i := 0; i < 36*7; i++ {
			date := start.AddDate(0, 0, i)
			count := activityCount(date)
			p.Cells = append(p.Cells, Cell{date.Format("02/01/2006") + fmt.Sprintf(" %d plays", count), count, fmt.Sprintf("%.3f", 0.1+float64(count)/100*0.9), date.After(today), i / 7})
		}
		if err := previewTemplates.ExecuteTemplate(w, "page.html", p); err != nil {
			t.Error(err)
		}
	})
	admin, err := adminui.New(s, adminui.Options{DisableRateLimit: true, Settings: referenceSettings{}})
	if err != nil {
		t.Fatal(err)
	}
	type entry struct {
		Name, Route string
		Status      int
	}
	var entries []entry
	volatile := regexp.MustCompile(`(name="(?:csrf_token|submission_token|plan)" value=")[^"]*(")`)
	// Only volatile form tokens are masked. The stored HTML otherwise retains all bytes.
	save := func(name, route string, h http.Handler, cookie *http.Cookie) {
		t.Helper()
		r := httptest.NewRequest("GET", route, nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", route, w.Code, w.Body.String())
		}
		body := volatile.ReplaceAllString(w.Body.String(), `${1}REFERENCE_TOKEN${2}`)
		write(name+".html", []byte(body))
		entries = append(entries, entry{name, route, w.Code})
	}
	for name, route := range map[string]string{"profiles": "/", "home": "/alpha?period=all_time&tz=UTC", "artists": "/alpha/chart/top/artists?period=all_time&tz=UTC", "albums": "/alpha/chart/top/albums?period=all_time&tz=UTC", "tracks": "/alpha/chart/top/tracks?period=all_time&tz=UTC", "listens": "/alpha/listens?period=all_time&tz=UTC", "artist": "/alpha/artist/1?tz=UTC", "album": "/alpha/album/1?tz=UTC", "track": "/alpha/track/1?tz=UTC", "rewind": "/alpha/rewind?year=2024&month=5&tz=UTC", "rewind-year": "/alpha/rewind?year=2024&tz=UTC", "empty": "/empty?tz=UTC"} {
		save("public/"+name, route, live, nil)
	}
	// Preview calendar cells use the same fixed date as the public fixture.
	save("preview/profiles", "/", preview, nil)
	save("preview/home", "/demo?period=all_time", preview, nil)
	r := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	admin.ServeHTTP(w, r)
	csrf := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if len(csrf) != 2 {
		t.Fatal("login csrf missing")
	}
	save("admin/login", "/admin", admin, nil)
	form := url.Values{"csrf_token": {csrf[1]}, "username": {"reference-admin"}, "password": {"fictional-password"}, "next": {"/admin"}}
	r = httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range w.Result().Cookies() {
		r.AddCookie(c)
	}
	w = httptest.NewRecorder()
	admin.ServeHTTP(w, r)
	if w.Code != 303 {
		t.Fatalf("fictional login: %d", w.Code)
	}
	var session *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == adminweb.SessionCookie && c.Value != "" && c.MaxAge >= 0 {
			session = c
		}
	}
	if session == nil {
		t.Fatal("session missing")
	}
	for name, route := range map[string]string{"profiles": "/admin?profile=1", "account": "/admin?section=account", "settings": "/admin/settings", "artist": "/alpha/manage/artist/1", "album": "/alpha/manage/album/1", "track": "/alpha/manage/track/1", "merge": "/alpha/manage/track/1/review?action=merge&to=2", "delete": "/admin/profile/1/delete-review"} {
		save("admin/"+name, route, admin, session)
	}
	if err := fs.WalkDir(files, "static", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		b, e := files.ReadFile(p)
		if e != nil {
			return e
		}
		write("web-assets/"+strings.TrimPrefix(p, "static/"), b)
		if strings.HasPrefix(p, "static/public/") {
			write(strings.TrimPrefix(p, "static/"), b)
		}
		write("preview-assets/"+strings.TrimPrefix(p, "static/"), b)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	avatar, err := files.ReadFile("static/avatar.webp")
	if err != nil {
		t.Fatal(err)
	}
	write("image/preview-photo/128x128.webp", avatar)
	mux := http.NewServeMux()
	bindAssets(mux)
	for _, route := range []string{"/preview-assets/themes.css", "/preview-assets/themes.json", "/preview-assets/cover/0.svg", "/preview-assets/cover/1.svg", "/preview-assets/cover/2.svg", "/preview-assets/cover/3.svg", "/preview-assets/cover/4.svg"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 200 {
			t.Fatal(route)
		}
		write(strings.TrimPrefix(route, "/"), w.Body.Bytes())
		if strings.HasSuffix(route, ".css") {
			write("web-assets/themes.css", w.Body.Bytes())
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	write("routes.json", data)
}
