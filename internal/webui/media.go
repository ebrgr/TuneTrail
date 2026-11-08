package webui

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"github.com/google/uuid"
)

func isMediaKind(kind string) bool { return kind == "artist" || kind == "album" || kind == "track" }

func (h *Pages) serveMedia(w http.ResponseWriter, r *http.Request, p LivePage, parts []string) {
	if len(parts) != 3 {
		h.fail(w, p, 404)
		return
	}
	id, err := strconv.ParseInt(parts[2], 10, 32)
	if err != nil || id <= 0 {
		h.fail(w, p, 400)
		return
	}
	// These pages have all-time statistics. Do not imply that query filters
	// control their header or replace the entity selected by the route.
	for key, values := range r.URL.Query() {
		if len(values) != 1 || key != "tz" && !(key == "period" && values[0] == "all_time") {
			h.fail(w, p, 400)
			return
		}
	}
	loc := webquery.Timezone(r)
	p.Options.Timeframe = db.Timeframe{Period: db.PeriodAllTime, Timezone: loc}
	p.Timezone, p.Period, p.Section = loc.String(), "all_time", "media"
	if err := h.loadMedia(r.Context(), &p, parts[1], int32(id)); err != nil {
		status := 500
		if errors.Is(err, db.ErrNotFound) {
			status = 404
		}
		log.Printf("HTML media %s/%d failed: %v", parts[1], id, err)
		h.fail(w, p, status)
		return
	}
	// A selected listen returns to the corresponding filtered personal history.
	returnQuery := url.Values{"period": {"all_time"}, parts[1] + "_id": {strconv.FormatInt(id, 10)}, "tz": {loc.String()}}
	p.CurrentURL = "/" + url.PathEscape(p.Profile.Slug) + "/listens?" + returnQuery.Encode()
	p.Title = p.Media.Title
	h.render(w, p)
}

func (h *Pages) loadMedia(ctx context.Context, p *LivePage, kind string, id int32) error {
	slug := p.Profile.Slug
	m := &MediaPage{Kind: kind}
	var first int64
	var mbid *uuid.UUID
	switch kind {
	case "artist":
		a, err := h.store.GetArtist(ctx, db.GetArtistOpts{ID: id})
		if err != nil {
			return err
		}
		m.Item, m.Type, m.TimeListened, first, mbid = artistItem(slug, a, a.AllTimeRank), "Artist", a.TimeListened, a.FirstListen, a.MbzID
	case "album":
		a, err := h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
		if err != nil {
			return err
		}
		m.Item, m.Type, m.TimeListened, first, mbid = albumItem(slug, a, a.AllTimeRank), "Album", a.TimeListened, a.FirstListen, a.MbzID
	case "track":
		t, err := h.store.GetTrack(ctx, db.GetTrackOpts{ID: id})
		if err != nil {
			return err
		}
		a, err := h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: t.AlbumID})
		if err != nil {
			return fmt.Errorf("album for track %d unavailable: %v", id, err)
		}
		m.Item, m.Type, m.TimeListened, first, mbid = trackItem(slug, t, t.AllTimeRank), "Track", t.TimeListened, t.FirstListen, t.MbzID
		m.Album = &Link{a.Title, mediaURL(slug, "album", a.ID)}
	}
	m.MinutesListened = m.TimeListened / 60
	if first > 0 {
		date := time.Unix(first, 0).In(p.Options.Timeframe.Timezone)
		m.FirstListen = &date
	}
	if mbid != nil && *mbid != uuid.Nil {
		entity := map[string]string{"artist": "artist", "album": "release", "track": "recording"}[kind]
		m.MusicBrainzURL = "https://musicbrainz.org/" + entity + "/" + mbid.String()
	}
	p.Media = m
	fail := func(block string, err error) bool {
		if err == nil {
			return false
		}
		p.Errors[block] = true
		log.Printf("HTML media %s/%d %s failed: %v", kind, id, block, err)
		return true
	}
	o := db.GetItemsOpts{Page: 1, Limit: 11, Timeframe: p.Options.Timeframe}
	q := url.Values{"period": {"all_time"}, kind + "_id": {strconv.FormatInt(int64(id), 10)}}
	if tz := p.Params.Get("tz"); tz != "" {
		q.Set("tz", tz)
	}
	switch kind {
	case "artist":
		o.ArtistID = int(id)
	case "album":
		o.AlbumID = int(id)
	case "track":
		o.TrackID = int(id)
	}
	activityOpts := db.ListenActivityOpts{Timezone: p.Options.Timeframe.Timezone, ArtistID: int32(o.ArtistID), AlbumID: int32(o.AlbumID), TrackID: int32(o.TrackID)}
	target := map[string]int64{"artist": 20, "album": 10, "track": 5}[kind]
	var err error
	p.Activity, err = readActivity(ctx, h.store, activityOpts, p.WeekStart, target, h.options.Now())
	fail("activity", err)
	interestOpts := db.GetInterestOpts{Buckets: db.DefaultInterestBuckets, ArtistID: int32(o.ArtistID), AlbumID: int32(o.AlbumID), TrackID: int32(o.TrackID)}
	interest, err := h.store.GetInterest(ctx, interestOpts)
	if !fail("interest", err) {
		m.Interest, err = buildInterest(interest, p.Options.Timeframe.Timezone, fmt.Sprintf("interest-%d-%s-%d", p.Profile.ID, kind, id))
		fail("interest", err)
	}
	m.HistoryURL = queryURL("/"+slug+"/listens", q)
	if kind != "track" {
		topOpts := o
		topOpts.Limit = 8
		items, _, err := h.ranking(ctx, slug, "tracks", topOpts)
		m.TopTracks = Card{Title: "Top tracks", URL: queryURL("/"+slug+"/chart/top/tracks", q), Items: items, Error: fail("tracks", err)}
	}
	recent, err := h.store.GetListensPaginated(ctx, o)
	if !fail("listens", err) {
		p.Listens = h.listens(slug, recent.Items, p.Options.Timeframe.Timezone)
		p.HistoryMore = recent.HasNextPage
	}
	playing, err := listening.ReadNowPlaying(ctx, h.store)
	if !fail("playing", err) && playing.CurrentlyPlaying && mediaMatchesPlaying(kind, id, playing.Track) {
		item := trackItem(slug, &playing.Track, 0)
		p.Playing = &item
	}
	if kind == "artist" {
		albumOpts := o
		albumOpts.Limit = 99
		items, pager, err := h.ranking(ctx, slug, "albums", albumOpts)
		if !fail("albums", err) {
			m.Albums = items
			m.AlbumsMore = pager.Next != ""
		}
		m.AlbumsURL = queryURL("/"+slug+"/chart/top/albums", q)
	}
	return nil
}

func mediaMatchesPlaying(kind string, id int32, track models.Track) bool {
	switch kind {
	case "track":
		return track.ID == id
	case "album":
		return track.AlbumID == id
	case "artist":
		for _, artist := range track.Artists {
			if artist.ID == id {
				return true
			}
		}
	}
	return false
}
