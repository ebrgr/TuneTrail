package webui

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"log"
)

func (h *Pages) loadHome(ctx context.Context, p *LivePage) {
	fail := func(key string, err error) bool {
		if err == nil {
			return false
		}
		p.Errors[key] = true
		log.Printf("HTML home %s failed: %v", key, err)
		return true
	}
	loc := p.Options.Timeframe.Timezone
	all := db.Timeframe{Period: db.PeriodAllTime, Timezone: loc}
	var err error
	p.Stats, err = listening.ReadStats(ctx, h.store, all)
	fail("stats", err)
	p.Activity, err = readActivity(ctx, h.store, db.ListenActivityOpts{Timezone: loc}, p.WeekStart, 100, h.options.Now())
	fail("activity", err)
	for _, kind := range []string{"artists", "albums", "tracks"} {
		o := db.GetItemsOpts{Page: 1, Limit: 5, Timeframe: p.Options.Timeframe}
		if kind == "tracks" {
			o.Limit = 10
		}
		items, _, err := h.ranking(ctx, p.Profile.Slug, kind, o)
		bad := fail(kind, err)
		q := "?period=" + p.Period
		p.Cards = append(p.Cards, Card{Title: "Top " + kind, URL: "/" + p.Profile.Slug + "/chart/top/" + kind + q, Items: items, Error: bad})
	}
	recent, err := h.store.GetListensPaginated(ctx, db.GetItemsOpts{Page: 1, Limit: 28, Timeframe: all})
	if !fail("listens", err) {
		p.Listens = h.listens(p.Profile.Slug, recent.Items, loc)
		p.HistoryMore = recent.HasNextPage
	}
	playing, err := listening.ReadNowPlaying(ctx, h.store)
	if !fail("playing", err) && playing.CurrentlyPlaying {
		item := trackItem(p.Profile.Slug, &playing.Track, 0)
		p.Playing = &item
	}
}
