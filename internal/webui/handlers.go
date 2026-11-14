package webui

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/version"
	"github.com/ebrgr/TuneTrail/internal/webquery"
)

// ReadStore is deliberately independent of the SQLite implementation.
type ReadStore interface {
	listening.StatsStore
	db.ProfileStore
}
type Options struct {
	Languages      *i18n.Catalog
	ForcedTimezone bool
	DefaultTheme   string
	Now            func() time.Time
}
type Pages struct {
	store     ReadStore
	options   Options
	templates *template.Template
}

func NewHandler(store ReadStore, options Options) (*Pages, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	if _, ok := palettes[options.DefaultTheme]; !ok {
		options.DefaultTheme = "yuu"
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Pages{store, options, tmpl}, nil
}
func (h *Pages) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	w = h.options.Languages.Wrap(w, r)
	p := LivePage{Page: Page{Locale: i18n.ViewOf(w), Theme: h.options.DefaultTheme, Version: version.Current, Live: true, ForcedTimezone: h.options.ForcedTimezone, SwitchLinks: map[string]string{}}, CurrentURL: r.URL.RequestURI(), Title: "Listening profiles", Section: "profiles", Status: 200, Errors: map[string]bool{}, Params: cloneQuery(r.URL.Query())}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if r.URL.Path != "/" {
		profile, err := h.store.GetListeningProfile(r.Context(), parts[0])
		if err != nil {
			status := 500
			if errors.Is(err, db.ErrNotFound) {
				status = 404
			}
			h.fail(w, p, status)
			return
		}
		public := publicProfile(*profile)
		p.Profile = &public
		r = r.WithContext(db.WithProfile(r.Context(), profile.ID))
		p.Section = "home"
		if len(parts) > 1 {
			p.Section = strings.Join(parts[1:], "/")
		}
	}
	all, err := h.store.ListListeningProfiles(r.Context())
	if err != nil {
		h.fail(w, p, 500)
		return
	}
	for _, profile := range all {
		p.Profiles = append(p.Profiles, publicProfile(profile))
		p.SwitchLinks[profile.Slug] = switchURL(profile.Slug, r.URL.Path, r.URL.Query())
	}
	for _, name := range themeNames {
		p.Themes = append(p.Themes, ThemeOption{name, palettes[name]})
	}
	if p.Profile == nil {
		p.Selection = true
		p.Next = safeNext(r.URL.Query().Get("next"))
		h.render(w, p)
		return
	}
	// Renamed from koito-weekstart in ISSUE-111; the browser rewrites it on the next visit.
	if c, e := r.Cookie("tunetrail-weekstart"); e == nil {
		if n, e := strconv.Atoi(c.Value); e == nil && n >= 0 && n <= 6 {
			p.WeekStart = n
		}
	}
	// Dispatch scoped media pages after resolving the profile.
	if len(parts) >= 2 && isMediaKind(parts[1]) {
		h.serveMedia(w, r, p, parts)
		return
	}
	// Rewind shares its summary service with the JSON API.
	if p.Section == "rewind" {
		h.serveRewind(w, r, p)
		return
	}
	var queryErr error
	p.Options, p.Params, queryErr = webquery.HTML(r, p.Section == "home")
	if queryErr != nil {
		h.fail(w, p, 400)
		return
	}
	// History links carry normalized supported filters, including calendar weeks.
	returnURL := *r.URL
	returnParams := cloneQuery(p.Params)
	for key, values := range returnParams {
		switch key {
		case "period", "year", "month", "week", "page", "limit", "artist_id", "album_id", "track_id", "tz", "show_time":
			if len(values) != 1 || values[0] == "" {
				returnParams.Del(key)
			}
		default:
			returnParams.Del(key)
		}
	}
	returnURL.RawQuery = returnParams.Encode()
	p.CurrentURL = returnURL.RequestURI()
	// Only filters supported by this list are accepted. Multiple media filters
	// retain the store's established precedence, rather than implying intersection.
	if p.Section == "chart/top/artists" && (p.Options.ArtistID > 0 || p.Options.AlbumID > 0 || p.Options.TrackID > 0) || p.Section == "chart/top/albums" && (p.Options.AlbumID > 0 || p.Options.TrackID > 0) || p.Section == "chart/top/tracks" && p.Options.TrackID > 0 {
		h.fail(w, p, 400)
		return
	}
	p.Period = string(p.Options.Timeframe.Period)
	p.Timezone = p.Options.Timeframe.Timezone.String()
	ctx := r.Context()
	slug := p.Profile.Slug
	switch p.Section {
	case "home":
		p.Title = "Koito"
		h.loadHome(ctx, &p)
	case "chart/top/artists", "chart/top/albums", "chart/top/tracks":
		kind := strings.TrimPrefix(p.Section, "chart/top/")
		p.Title = "Top " + strings.ToUpper(kind[:1]) + kind[1:]
		p.Items, p.Pager, err = h.ranking(ctx, slug, kind, p.Options)
	case "search":
		p.Title = "Search"
		p.Search = strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(p.Search)) > 200 {
			h.fail(w, p, 400)
			return
		}
		p.Items, err = h.search(ctx, slug, p.Search)
	case "listens":
		p.Title = "Last Played"
		var result *db.PaginatedResponse[*models.Listen]
		result, err = h.store.GetListensPaginated(ctx, p.Options)
		if err == nil {
			p.Listens = h.listens(slug, result.Items, p.Options.Timeframe.Timezone, p.Locale)
			p.Pager = pageInfo(result.CurrentPage, result.TotalCount, result.HasNextPage)
		}
	default:
		h.fail(w, p, 404)
		return
	}
	if err != nil {
		log.Printf("HTML %s read failed: %v", p.Section, err)
		h.fail(w, p, 500)
		return
	}
	path := r.URL.Path
	if p.Pager.Current > 1 {
		q := cloneQuery(p.Params)
		q.Set("page", strconv.Itoa(p.Pager.Current-1))
		p.Pager.Previous = queryURL(path, q)
	}
	if p.Pager.Next != "" {
		q := cloneQuery(p.Params)
		q.Set("page", strconv.Itoa(p.Pager.Current+1))
		p.Pager.Next = queryURL(path, q)
	}
	for _, period := range []string{"day", "week", "month", "year", "all_time"} {
		q := cloneQuery(p.Params)
		for _, key := range []string{"year", "month", "week", "from", "to", "page"} {
			q.Del(key)
		}
		q.Set("period", period)
		p.Periods = append(p.Periods, PeriodLink{Key: period, Label: translated(p.Locale, "period."+period, period), URL: queryURL(path, q)})
	}
	p.DateRange = webquery.RangeLabel(p.Options.Timeframe)
	first, firstErr := h.store.GetFirstListenUnix(ctx)
	if firstErr != nil && !errors.Is(firstErr, sql.ErrNoRows) {
		log.Printf("HTML first activity failed: %v", firstErr)
	}
	now := h.options.Now().In(p.Options.Timeframe.Timezone)
	lower := now.Year()
	if firstErr == nil && first > 0 {
		lower = time.Unix(first, 0).In(p.Options.Timeframe.Timezone).Year()
	}
	for y := now.Year(); y >= lower && y >= 1; y-- {
		p.Years = append(p.Years, y)
	}
	h.render(w, p)
}
func (h *Pages) render(w http.ResponseWriter, p LivePage) {
	// Page titles are presentation labels; media titles and slugs stay untouched.
	if p.Error != "" {
		key := map[int]string{400: "error.bad_request", 404: "error.not_found", 500: "error.internal"}[p.Status]
		if key != "" {
			p.Error = translated(p.Locale, key, p.Error)
			p.Title = p.Error
		}
	} else {
		key := map[string]string{"profiles": "page.profiles", "search": "page.search", "listens": "page.listens", "chart/top/artists": "page.top_artists", "chart/top/albums": "page.top_albums", "chart/top/tracks": "page.top_tracks"}[p.Section]
		if key != "" {
			p.Title = translated(p.Locale, key, p.Title)
		}
	}
	var b bytes.Buffer
	tmpl, err := localizedTemplates(h.templates, p.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&b, "live.html", p)
	}
	if err != nil {
		log.Printf("HTML render failed: %v", err)
		http.Error(w, "Unable to render page", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(p.Status)
	_, _ = b.WriteTo(w)
}
func (h *Pages) fail(w http.ResponseWriter, p LivePage, status int) {
	p.Status = status
	p.Error = http.StatusText(status)
	p.Title = p.Error
	h.render(w, p)
}
func pageInfo(page int32, count int64, next bool) Pager {
	p := Pager{Current: int(page), Count: count}
	if next {
		p.Next = "next"
	}
	return p
}
func (h *Pages) ranking(ctx context.Context, slug, kind string, o db.GetItemsOpts) ([]Item, Pager, error) {
	items := []Item{}
	switch kind {
	case "artists":
		r, e := h.store.GetTopArtistsPaginated(ctx, o)
		if e != nil {
			return nil, Pager{}, e
		}
		for _, v := range r.Items {
			items = append(items, artistItem(slug, v.Item, v.Rank))
		}
		return items, pageInfo(r.CurrentPage, r.TotalCount, r.HasNextPage), nil
	case "albums":
		r, e := h.store.GetTopAlbumsPaginated(ctx, o)
		if e != nil {
			return nil, Pager{}, e
		}
		for _, v := range r.Items {
			items = append(items, albumItem(slug, v.Item, v.Rank))
		}
		return items, pageInfo(r.CurrentPage, r.TotalCount, r.HasNextPage), nil
	case "tracks":
		r, e := h.store.GetTopTracksPaginated(ctx, o)
		if e != nil {
			return nil, Pager{}, e
		}
		for _, v := range r.Items {
			items = append(items, trackItem(slug, v.Item, v.Rank))
		}
		return items, pageInfo(r.CurrentPage, r.TotalCount, r.HasNextPage), nil
	}
	return nil, Pager{}, fmt.Errorf("unsupported ranking")
}
func (h *Pages) listens(slug string, items []*models.Listen, loc *time.Location, locale *i18n.View) []ListenView {
	out := make([]ListenView, 0, len(items))
	for _, v := range items {
		out = append(out, ListenView{Item: Item{ID: v.Track.ID, Title: v.Track.Title, URL: mediaURL(slug, "track", v.Track.ID), Image: v.Track.Image, Artists: itemLinks(slug, v.Track.Artists)}, Time: v.Time.In(loc), Unix: v.Time.Unix(), Relative: localizedRelativeTime(locale, h.options.Now(), v.Time.In(loc))})
	}
	return out
}
func relativeTime(now, t time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		return "just now"
	}
	if d >= 24*time.Hour {
		return t.Format("02/01/06 15:04")
	}
	n := int(d / time.Minute)
	unit := "minute"
	if n >= 60 {
		n /= 60
		unit = "hour"
	}
	if n != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s ago", n, unit)
}

// localizedRelativeTime preserves the original elapsed-time thresholds.
func localizedRelativeTime(locale *i18n.View, now, t time.Time) string {
	if locale == nil {
		return relativeTime(now, t)
	}
	d := now.Sub(t)
	if d < time.Minute {
		return locale.T("time.just_now")
	}
	if d >= 24*time.Hour {
		return locale.Date(t) + t.Format(" 15:04")
	}
	if d >= time.Hour {
		return locale.Quantity("quantity.ago_hours", int64(d/time.Hour))
	}
	return locale.Quantity("quantity.ago_minutes", int64(d/time.Minute))
}
func safeNext(raw string) string {
	u, e := url.Parse(raw)
	if e != nil || !strings.HasPrefix(raw, "/") || u.IsAbs() || u.Host != "" || strings.HasPrefix(raw, "//") {
		return ""
	}
	p := strings.Trim(u.Path, "/")
	if p == "listens" || p == "rewind" || p == "chart/top/artists" || p == "chart/top/albums" || p == "chart/top/tracks" {
		return raw
	}
	parts := strings.Split(p, "/")
	if len(parts) == 2 && (parts[0] == "artist" || parts[0] == "album" || parts[0] == "track") {
		id, e := strconv.ParseInt(parts[1], 10, 32)
		if e == nil && id > 0 {
			return raw
		}
	}
	return ""
}
